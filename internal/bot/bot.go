package bot

import (
	"context"
	"errors"
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

func New(cfg *config.Config, logger *logger.Logger) (*Bot, error) {
	if cfg == nil || logger == nil || cfg.Discord.Token == "" {
		return nil, errors.New("invalid configuration")
	}

	db, err := database.NewMongoDB(cfg)
	if err != nil {
		return nil, err
	}

	return &Bot{
		config:   cfg,
		logger:   logger,
		db:       db,
		sessions: make([]*discordgo.Session, 0),
	}, nil
}

func (b *Bot) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b == nil || b.config == nil {
		return errors.New("invalid bot instance")
	}

	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		if b.config.Discord.Sharding.Enabled {
			done <- b.startSharded()
		} else {
			done <- b.startSingle()
		}
	}()

	select {
	case err := <-done:
		return err
	case <-startCtx.Done():
		return errors.New("bot startup timed out")
	}
}

func (b *Bot) startSingle() error {
	session, err := discordgo.New("Bot " + b.config.Discord.Token)
	if err != nil {
		return err
	}

	b.sessions = []*discordgo.Session{session}
	return b.setupSession(session, 0, 1)
}

func (b *Bot) startSharded() error {
	totalShards := b.config.Discord.Sharding.TotalShards
	if totalShards <= 0 {
		return errors.New("invalid shard count")
	}

	b.sessions = make([]*discordgo.Session, totalShards)
	var wg sync.WaitGroup
	errChan := make(chan error, totalShards)

	for i := 0; i < totalShards; i++ {
		wg.Add(1)
		go func(shardID int) {
			defer wg.Done()
			session, err := discordgo.New("Bot " + b.config.Discord.Token)
			if err != nil {
				errChan <- err
				return
			}

			b.sessions[shardID] = session
			if err := b.setupSession(session, shardID, totalShards); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) setupSession(session *discordgo.Session, shardID, totalShards int) error {
	if session == nil {
		return errors.New("invalid session")
	}

	session.ShardID = shardID
	session.ShardCount = totalShards
	session.Identify.Intents = discordgo.IntentsAll

	if shardID == 0 {
		b.cmdHandler = commands.NewHandler(session, b.config, b.logger)
		b.eventHandler = events.NewHandler(session, b.config, b.logger)
		b.guildHandler = guild.NewHandler(b.db, b.logger)

		if b.cmdHandler == nil || b.eventHandler == nil || b.guildHandler == nil {
			return errors.New("failed to initialize handlers")
		}

		admin.SetDatabase(b.db)

		if err := b.cmdHandler.LoadCommands(); err != nil {
			return err
		}

		if err := b.eventHandler.LoadEvents(); err != nil {
			return err
		}
	}

	session.AddHandler(b.cmdHandler.HandleCommand)
	session.AddHandler(b.guildHandler.HandleGuildCreate)
	session.AddHandler(b.guildHandler.HandleGuildDelete)
	session.AddHandler(admin.HandleConfigButton)

	return session.Open()
}

func (b *Bot) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b == nil {
		return errors.New("invalid bot instance")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(b.sessions)+2)

	if b.cmdHandler != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.cmdHandler.DeleteCommands(); err != nil {
				errChan <- err
			}
		}()
	}

	if b.db != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.db.Close(); err != nil {
				errChan <- err
			}
		}()
	}

	for _, session := range b.sessions {
		if session != nil {
			wg.Add(1)
			go func(s *discordgo.Session) {
				defer wg.Done()
				if err := s.Close(); err != nil {
					errChan <- err
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
		return errors[0]
	}

	return nil
}
