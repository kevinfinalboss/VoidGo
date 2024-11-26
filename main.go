package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kevinfinalboss/Void/api/server"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/bot"
	"github.com/kevinfinalboss/Void/internal/logger"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	logger := logger.New(cfg.Logger)
	if logger == nil {
		log.Fatal("Failed to initialize logger")
	}

	srv := server.NewServer(cfg)
	if srv == nil {
		logger.Fatal("Failed to create server")
	}

	srv.SetupRoutes()
	go func() {
		logger.Info("Starting HTTP server...")
		if err := srv.Start(); err != nil {
			logger.Error("HTTP server error: " + err.Error())
		}
	}()

	logger.Info("Initializing bot...")
	discordBot, err := bot.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create bot: " + err.Error())
	}

	logger.Info("Starting bot...")
	if err := discordBot.Start(); err != nil {
		logger.Fatal("Failed to start bot: " + err.Error())
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	<-sc

	logger.Info("Shutting down...")
	if err := discordBot.Stop(); err != nil {
		logger.Error("Error during shutdown: " + err.Error())
	}
}
