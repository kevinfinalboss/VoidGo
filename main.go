package main

import (
	"context"
	"log"
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

	srv := server.NewServer(cfg)
	if srv == nil {
		logger.Fatal("Failed to create server")
	}

	srv.SetupRoutes()
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting HTTP server...")

		serverCtx, serverCancel := context.WithCancel(mainCtx)
		defer serverCancel()

		go func() {
			<-serverCtx.Done()
		}()

		if err := srv.Start(); err != nil {
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
			<-botCtx.Done()
		}()

		if err := discordBot.Start(); err != nil {
			errChan <- err
			mainCancel()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	select {
	case sig := <-sigChan:
		logger.Info("Received signal: " + sig.String())
		mainCancel()
	case err := <-errChan:
		logger.Error("Error during execution: " + err.Error())
		mainCancel()
	case <-mainCtx.Done():
		logger.Info("Context cancelled, initiating shutdown...")
	}

	logger.Info("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	shutdownChan := make(chan struct{})
	go func() {
		defer close(shutdownChan)
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-shutdownCtx.Done():
			logger.Error("Timeout waiting for goroutines to finish")
		case <-done:
			logger.Info("All goroutines finished successfully")
		}
		if err := discordBot.Stop(); err != nil {
			logger.Error("Error during bot shutdown: " + err.Error())
		}
	}()

	select {
	case <-shutdownCtx.Done():
		logger.Error("Global shutdown timed out")
	case <-shutdownChan:
		logger.Info("Shutdown completed successfully")
	}
}
