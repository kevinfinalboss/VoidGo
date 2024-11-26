package bot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/commands/admin"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/events/guild"
	"github.com/kevinfinalboss/Void/internal/commands"
	"github.com/kevinfinalboss/Void/internal/database"
	"github.com/kevinfinalboss/Void/internal/events"
	"github.com/kevinfinalboss/Void/internal/logger"
)

type Bot struct {
	sessions     []*discordgo.Session
	config       *config.Config
	logger       *logger.Logger
	cmdHandler   *commands.Handler
	eventHandler *events.Handler
	db           *database.MongoDB
	guildHandler *guild.Handler
	mu           sync.RWMutex
}

func New(cfg *config.Config, l *logger.Logger) (*Bot, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if l == nil {
		return nil, errors.New("logger cannot be nil")
	}
	if cfg.Discord.Token == "" {
		return nil, errors.New("discord token is required")
	}

	db, err := database.NewMongoDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	guildHandler := guild.NewHandler(db, l)
	if guildHandler == nil {
		if db != nil {
			db.Close()
		}
		return nil, errors.New("failed to create guild handler")
	}

	bot := &Bot{
		config:       cfg,
		logger:       l,
		db:           db,
		sessions:     make([]*discordgo.Session, 0),
		guildHandler: guildHandler,
	}

	return bot, nil
}

func (b *Bot) setupHandlers(session *discordgo.Session) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	// Verificações de dependências necessárias
	if b.config == nil || b.logger == nil || b.db == nil || b.guildHandler == nil {
		return errors.New("bot dependencies not properly initialized")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Inicializar command handler
	if b.cmdHandler == nil {
		cmdHandler := commands.NewHandler(session, b.config, b.logger)
		if cmdHandler == nil {
			return errors.New("failed to create command handler")
		}
		b.cmdHandler = cmdHandler
	}

	// Inicializar event handler
	if b.eventHandler == nil {
		eventHandler := events.NewHandler(session, b.config, b.logger)
		if eventHandler == nil {
			return errors.New("failed to create event handler")
		}
		b.eventHandler = eventHandler
	}

	// Configurar database para admin
	admin.SetDatabase(b.db)

	// Adicionar handlers
	session.AddHandler(b.cmdHandler.HandleCommand)
	session.AddHandler(b.guildHandler.HandleGuildCreate)
	session.AddHandler(b.guildHandler.HandleGuildDelete)
	session.AddHandler(admin.HandleConfigButton)

	// Carregar comandos e eventos
	if err := b.cmdHandler.LoadCommands(); err != nil {
		return fmt.Errorf("failed to load commands: %v", err)
	}

	if err := b.eventHandler.LoadEvents(); err != nil {
		return fmt.Errorf("failed to load events: %v", err)
	}

	return nil
}

func (b *Bot) Start() error {
	if b == nil {
		return errors.New("bot instance is nil")
	}

	b.mu.Lock()
	if b.config == nil {
		b.mu.Unlock()
		return errors.New("bot configuration is nil")
	}
	isSharded := b.config.Discord.Sharding.Enabled
	b.mu.Unlock()

	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		if isSharded {
			errChan <- b.startSharded()
		} else {
			errChan <- b.startSingle()
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-startCtx.Done():
		return errors.New("bot startup timed out")
	}
}

func (b *Bot) startSingle() error {
	if b == nil || b.config == nil {
		return errors.New("invalid bot state")
	}

	session, err := discordgo.New("Bot " + b.config.Discord.Token)
	if err != nil {
		return fmt.Errorf("failed to create discord session: %v", err)
	}

	b.mu.Lock()
	b.sessions = []*discordgo.Session{session}
	b.mu.Unlock()

	if err := b.setupSession(session, 0, 1); err != nil {
		b.logger.Error("Failed to setup session: " + err.Error())
		return err
	}

	b.logger.Info("Bot started successfully in single mode")
	return nil
}

func (b *Bot) startSharded() error {
	if b == nil || b.config == nil {
		return errors.New("invalid bot state")
	}

	totalShards := b.config.Discord.Sharding.TotalShards
	if totalShards <= 0 {
		return errors.New("invalid shard count")
	}

	b.mu.Lock()
	b.sessions = make([]*discordgo.Session, totalShards)
	b.mu.Unlock()

	var wg sync.WaitGroup
	errChan := make(chan error, totalShards)

	for i := 0; i < totalShards; i++ {
		wg.Add(1)
		go func(shardID int) {
			defer wg.Done()
			session, err := discordgo.New("Bot " + b.config.Discord.Token)
			if err != nil {
				errChan <- fmt.Errorf("failed to create discord session for shard %d: %v", shardID, err)
				return
			}

			b.mu.Lock()
			b.sessions[shardID] = session
			b.mu.Unlock()

			if err := b.setupSession(session, shardID, totalShards); err != nil {
				errChan <- fmt.Errorf("failed to setup shard %d: %v", shardID, err)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors starting sharded mode: %v", errors)
	}

	b.logger.Info("Bot started successfully in sharded mode")
	return nil
}

func (b *Bot) setupSession(session *discordgo.Session, shardID, totalShards int) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	session.ShardID = shardID
	session.ShardCount = totalShards
	session.Identify.Intents = discordgo.IntentsAll

	if shardID == 0 {
		if err := b.setupHandlers(session); err != nil {
			return fmt.Errorf("failed to setup handlers: %v", err)
		}
	}

	if err := session.Open(); err != nil {
		return fmt.Errorf("failed to open session: %v", err)
	}

	return nil
}

func (b *Bot) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b == nil {
		return errors.New("bot instance is nil")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(b.sessions)+2)

	// Cleanup command handler
	if b.cmdHandler != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.cmdHandler.DeleteCommands(); err != nil {
				errChan <- fmt.Errorf("failed to delete commands: %v", err)
			}
		}()
	}

	// Cleanup database
	if b.db != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.db.Close(); err != nil {
				errChan <- fmt.Errorf("failed to close database: %v", err)
			}
		}()
	}

	// Close all sessions
	for _, session := range b.sessions {
		if session != nil {
			wg.Add(1)
			go func(s *discordgo.Session) {
				defer wg.Done()
				if err := s.Close(); err != nil {
					errChan <- fmt.Errorf("failed to close session: %v", err)
				}
			}(session)
		}
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}
