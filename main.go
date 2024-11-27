package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kevinfinalboss/Void/api/server"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/bot"
	"github.com/kevinfinalboss/Void/internal/logger"
)

func main() {
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	logger := logger.New(cfg.Logger)
	if logger == nil {
		log.Fatal("Failed to initialize logger")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 2)
	shutdownChan := make(chan struct{})

	srv := server.NewServer(cfg)
	if srv == nil {
		logger.Fatal("Failed to create server")
	}

	srv.SetupRoutes()
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting HTTP server...")

		go func() {
			select {
			case <-mainCtx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := srv.Shutdown(shutdownCtx); err != nil {
					logger.Error("Server shutdown error: " + err.Error())
				}
			case <-shutdownChan:
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := srv.Shutdown(shutdownCtx); err != nil {
					logger.Error("Server shutdown error: " + err.Error())
				}
			}
		}()

		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- err
			mainCancel()
		}
	}()

	discordBot, err := bot.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create bot: " + err.Error())
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting bot...")

		botCtx, botCancel := context.WithCancel(mainCtx)
		defer botCancel()

		go func() {
			select {
			case <-botCtx.Done():
			case <-shutdownChan:
				botCancel()
			}
		}()

		if err := discordBot.Start(); err != nil {
			errChan <- err
			mainCancel()
			return
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	select {
	case sig := <-sigChan:
		logger.Info("Received signal: " + sig.String())
	case err := <-errChan:
		logger.Error("Error during execution: " + err.Error())
	}

	logger.Info("Initiating shutdown sequence...")
	close(shutdownChan)

	botShutdownCtx, botShutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer botShutdownCancel()

	botDone := make(chan struct{})
	go func() {
		if err := discordBot.Stop(); err != nil {
			logger.Error("Bot shutdown error: " + err.Error())
		}
		close(botDone)
	}()

	select {
	case <-botShutdownCtx.Done():
		logger.Error("Bot shutdown timed out")
	case <-botDone:
		logger.Info("Bot shutdown completed successfully")
	}

	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	wgTimeout := time.After(15 * time.Second)
	select {
	case <-wgDone:
		logger.Info("All goroutines finished successfully")
	case <-wgTimeout:
		logger.Error("Some goroutines did not finish in time")
	}

	logger.Info("Shutdown completed")
}
