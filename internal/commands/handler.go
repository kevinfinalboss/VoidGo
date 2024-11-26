package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/kevinfinalboss/Void/commands/all"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/logger"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

type Handler struct {
	commands     map[string]*types.Command
	session      *discordgo.Session
	config       *config.Config
	logger       *logger.Logger
	commandMutex sync.RWMutex
}

func NewHandler(s *discordgo.Session, cfg *config.Config, l *logger.Logger) *Handler {
	return &Handler{
		commands: make(map[string]*types.Command),
		session:  s,
		config:   cfg,
		logger:   l,
	}
}

func (h *Handler) LoadCommands() error {
	startTime := time.Now()
	h.logger.Info("Loading commands...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	commands := make([]*discordgo.ApplicationCommand, 0, len(registry.Commands))
	for name, cmd := range registry.Commands {
		command := &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     make([]*discordgo.ApplicationCommandOption, 0, len(cmd.Options)),
		}

		for _, opt := range cmd.Options {
			command.Options = append(command.Options, &discordgo.ApplicationCommandOption{
				Name:        opt.Name,
				Description: opt.Description,
				Type:        opt.Type,
				Required:    opt.Required,
				Choices:     opt.Choices,
			})
		}

		commands = append(commands, command)
		h.commandMutex.Lock()
		h.commands[name] = cmd
		h.commandMutex.Unlock()
	}

	done := make(chan error, 1)
	go func() {
		_, err := h.session.ApplicationCommandBulkOverwrite(
			h.config.Discord.ClientID,
			h.config.Discord.GuildID,
			commands,
		)
		done <- err
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("command registration timed out after %v", time.Since(startTime))
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to register commands: %v", err)
		}
		h.logger.Info(fmt.Sprintf("Successfully registered %d commands in %v", len(commands), time.Since(startTime)))
		return nil
	}
}

func (h *Handler) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		h.handleAutocomplete(s, i)
		return
	}

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	h.commandMutex.RLock()
	cmd, exists := h.commands[i.ApplicationCommandData().Name]
	h.commandMutex.RUnlock()

	if !exists {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run(s, i, h.config)
		close(done)
	}()

	select {
	case <-ctx.Done():
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Comando expirou. Tente novamente.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	case err := <-done:
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Ocorreu um erro ao executar o comando.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
}

func (h *Handler) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	h.commandMutex.RLock()
	cmd, exists := h.commands[i.ApplicationCommandData().Name]
	h.commandMutex.RUnlock()

	if !exists || cmd.AutoComplete == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		choices, err := cmd.AutoComplete(s, i)
		if err != nil {
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}
