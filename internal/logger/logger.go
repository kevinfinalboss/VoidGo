// internal/logger/logger.go
package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Level string
	File  string
}

type Logger struct {
	logger *log.Logger
}

func New(cfg struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}) *Logger {
	logConfig := Config{
		Level: cfg.Level,
		File:  cfg.File,
	}

	if logConfig.File == "" {
		logConfig.File = "logs/bot.log"
	}

	logConfig.File = strings.ReplaceAll(logConfig.File, "/", string(os.PathSeparator))

	if !filepath.IsAbs(logConfig.File) {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal("Failed to get working directory:", err)
		}
		logConfig.File = filepath.Join(dir, logConfig.File)
	}

	logDir := filepath.Dir(logConfig.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("Failed to create log directory:", err)
	}

	file, err := os.OpenFile(logConfig.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to open log file %s: %v", logConfig.File, err))
	}

	multiWriter := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	fileLogger := log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)

	multiWriter.Printf("Logger initialized. Log file: %s", logConfig.File)

	return &Logger{
		logger: fileLogger,
	}
}

func (l *Logger) Info(v ...interface{}) {
	message := fmt.Sprint(v...)
	l.logger.Printf("INFO: %s", message)
	log.Printf("INFO: %s", message)
}

func (l *Logger) Error(v ...interface{}) {
	message := fmt.Sprint(v...)
	l.logger.Printf("ERROR: %s", message)
	log.Printf("ERROR: %s", message)
}

func (l *Logger) Fatal(v ...interface{}) {
	message := fmt.Sprint(v...)
	l.logger.Printf("FATAL: %s", message)
	log.Fatalf("FATAL: %s", message)
}
