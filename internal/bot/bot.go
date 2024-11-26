package bot

import (
	"errors"

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
}

func New(cfg *config.Config, logger *logger.Logger) (*Bot, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	if cfg.Discord.Token == "" {
		return nil, errors.New("discord token is required")
	}

	db, err := database.NewMongoDB(cfg)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		config:   cfg,
		logger:   logger,
		db:       db,
		sessions: make([]*discordgo.Session, 0),
	}

	return bot, nil
}

func (b *Bot) Start() error {
	if b == nil {
		return errors.New("bot instance is nil")
	}
	if b.config == nil {
		return errors.New("bot config is nil")
	}

	b.logger.Info("Starting bot...")

	if b.config.Discord.Sharding.Enabled {
		return b.startSharded()
	}
	return b.startSingle()
}

func (b *Bot) startSingle() error {
	session, err := discordgo.New("Bot " + b.config.Discord.Token)
	if err != nil {
		return err
	}

	b.sessions = []*discordgo.Session{session}
	if err := b.setupSession(session, 0, 1); err != nil {
		b.logger.Error("Failed to setup session: " + err.Error())
		return err
	}

	b.logger.Info("Bot started successfully in single mode")
	return nil
}

func (b *Bot) startSharded() error {
	totalShards := b.config.Discord.Sharding.TotalShards
	if totalShards <= 0 {
		return errors.New("invalid total shards count")
	}

	b.sessions = make([]*discordgo.Session, totalShards)

	for i := 0; i < totalShards; i++ {
		session, err := discordgo.New("Bot " + b.config.Discord.Token)
		if err != nil {
			return err
		}

		b.sessions[i] = session
		if err := b.setupSession(session, i, totalShards); err != nil {
			b.logger.Error("Failed to setup shard " + string(i) + ": " + err.Error())
			return err
		}
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
		b.cmdHandler = commands.NewHandler(session, b.config, b.logger)
		if b.cmdHandler == nil {
			return errors.New("failed to create command handler")
		}

		b.eventHandler = events.NewHandler(session, b.config, b.logger)
		if b.eventHandler == nil {
			return errors.New("failed to create event handler")
		}

		b.guildHandler = guild.NewHandler(b.db, b.logger)
		if b.guildHandler == nil {
			return errors.New("failed to create guild handler")
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

	err := session.Open()
	if err != nil {
		return err
	}

	b.logger.Info("Session setup completed for shard " + string(shardID))
	return nil
}

func (b *Bot) Stop() error {
	if b == nil {
		return errors.New("bot instance is nil")
	}

	if b.cmdHandler != nil {
		if err := b.cmdHandler.DeleteCommands(); err != nil {
			b.logger.Error("Error deleting commands: " + err.Error())
		}
	}

	if b.db != nil {
		if err := b.db.Close(); err != nil {
			b.logger.Error("Error closing MongoDB connection: " + err.Error())
		}
	}

	for _, session := range b.sessions {
		if session != nil {
			if err := session.Close(); err != nil {
				b.logger.Error("Error closing session: " + err.Error())
			}
		}
	}

	b.logger.Info("Bot stopped successfully")
	return nil
}
