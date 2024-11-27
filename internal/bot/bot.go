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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbChan := make(chan *database.MongoDB)
	errChan := make(chan error)

	go func() {
		db, err := database.NewMongoDB(cfg)
		if err != nil {
			errChan <- fmt.Errorf("failed to initialize database: %v", err)
			return
		}
		dbChan <- db
	}()

	select {
	case err := <-errChan:
		return nil, err
	case db := <-dbChan:
		guildHandler := guild.NewHandler(db, l)
		if guildHandler == nil {
			db.Close()
			return nil, errors.New("failed to create guild handler")
		}

		return &Bot{
			config:       cfg,
			logger:       l,
			db:           db,
			sessions:     make([]*discordgo.Session, 0),
			guildHandler: guildHandler,
		}, nil
	case <-ctx.Done():
		return nil, errors.New("timeout initializing bot dependencies")
	}
}

func (b *Bot) setupHandlers(session *discordgo.Session) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if session == nil {
		return errors.New("session cannot be nil")
	}

	if b.config == nil || b.logger == nil || b.db == nil || b.guildHandler == nil {
		return errors.New("bot dependencies not properly initialized")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	errChan := make(chan error, 2)
	setupDone := make(chan struct{})

	go func() {
		defer close(setupDone)

		if b.cmdHandler == nil {
			cmdHandler := commands.NewHandler(session, b.config, b.logger)
			if cmdHandler == nil {
				errChan <- errors.New("failed to create command handler")
				return
			}
			b.cmdHandler = cmdHandler
		}

		if b.eventHandler == nil {
			eventHandler := events.NewHandler(session, b.config, b.logger)
			if eventHandler == nil {
				errChan <- errors.New("failed to create event handler")
				return
			}
			b.eventHandler = eventHandler
		}

		admin.SetDatabase(b.db)

		session.AddHandler(b.cmdHandler.HandleCommand)
		session.AddHandler(b.guildHandler.HandleGuildCreate)
		session.AddHandler(b.guildHandler.HandleGuildDelete)
		session.AddHandler(admin.HandleConfigButton)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			if err := b.cmdHandler.LoadCommands(); err != nil {
				errChan <- fmt.Errorf("failed to load commands: %v", err)
			}
		}()

		go func() {
			defer wg.Done()
			if err := b.eventHandler.LoadEvents(); err != nil {
				errChan <- fmt.Errorf("failed to load events: %v", err)
			}
		}()

		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return errors.New("setup handlers timed out")
	case err := <-errChan:
		return err
	case <-setupDone:
		return nil
	}
}

func (b *Bot) Start() error {
	startCtx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

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

	errChan := make(chan error, 1)
	startDone := make(chan struct{})

	go func() {
		defer close(startDone)
		if isSharded {
			errChan <- b.startSharded()
		} else {
			errChan <- b.startSingle()
		}
	}()

	select {
	case <-startCtx.Done():
		return errors.New("bot startup timed out")
	case err := <-errChan:
		return err
	case <-startDone:
		return nil
	}
}

func (b *Bot) startSingle() error {
	sessionCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

	setupDone := make(chan error, 1)
	go func() {
		setupDone <- b.setupSession(session, 0, 1)
	}()

	select {
	case <-sessionCtx.Done():
		return errors.New("session setup timed out")
	case err := <-setupDone:
		if err != nil {
			b.logger.Error("Failed to setup session: " + err.Error())
			return err
		}
		b.logger.Info("Bot started successfully in single mode")
		return nil
	}
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
	semaphore := make(chan struct{}, 5)

	for i := 0; i < totalShards; i++ {
		wg.Add(1)
		go func(shardID int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

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
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b == nil {
		return errors.New("bot instance is nil")
	}

	// Fecha as sessões primeiro
	var wg sync.WaitGroup
	sessionErrors := make(chan error, len(b.sessions))

	for _, session := range b.sessions {
		if session != nil {
			wg.Add(1)
			go func(s *discordgo.Session) {
				defer wg.Done()
				if err := s.Close(); err != nil {
					sessionErrors <- fmt.Errorf("failed to close session: %v", err)
				}
			}(session)
		}
	}

	// Aguarda o fechamento das sessões com timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return errors.New("session shutdown timed out")
	case <-done:
		// Continue com o fechamento do banco de dados
	}

	// Fecha o banco de dados por último
	if b.db != nil {
		if err := b.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %v", err)
		}
	}

	close(sessionErrors)

	var errors []error
	for err := range sessionErrors {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}
