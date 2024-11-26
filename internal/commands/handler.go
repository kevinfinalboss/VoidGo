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
	cooldowns    sync.Map
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	loadComplete := make(chan struct{})
	errChan := make(chan error, 1)

	go func() {
		defer close(loadComplete)
		defer close(errChan)

		commands := make([]*types.Command, 0, len(registry.Commands))
		nameMap := make(map[*types.Command]string)
		for name, cmd := range registry.Commands {
			commands = append(commands, cmd)
			nameMap[cmd] = name
		}

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 3)
		registrationErrors := make(chan error, len(commands))

		for _, cmd := range commands {
			select {
			case <-ctx.Done():
				errChan <- fmt.Errorf("command registration timed out")
				return
			default:
				wg.Add(1)
				go func(cmd *types.Command) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					name := nameMap[cmd]
					if err := h.registerCommand(name, cmd); err != nil {
						registrationErrors <- fmt.Errorf("failed to register %s: %v", name, err)
					} else {
						h.logger.Info(fmt.Sprintf("Registered command: %s", name))
					}
					time.Sleep(200 * time.Millisecond)
				}(cmd)
			}
		}

		wgDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(wgDone)
		}()

		select {
		case <-ctx.Done():
			errChan <- fmt.Errorf("command registration timed out")
		case <-wgDone:
			close(registrationErrors)
			var errors []error
			for err := range registrationErrors {
				errors = append(errors, err)
			}
			if len(errors) > 0 {
				errChan <- fmt.Errorf("failed to register some commands: %v", errors)
			}
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("command loading timed out after %v", time.Since(startTime))
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("error loading commands: %v", err)
		}
	case <-loadComplete:
		h.logger.Info(fmt.Sprintf("Successfully loaded all commands in %v", time.Since(startTime)))
	}

	return nil
}

func (h *Handler) registerCommand(name string, cmd *types.Command) error {
	options := make([]*discordgo.ApplicationCommandOption, 0, len(cmd.Options))
	for _, opt := range cmd.Options {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        opt.Name,
			Description: opt.Description,
			Type:        opt.Type,
			Required:    opt.Required,
			Choices:     opt.Choices,
		})
	}

	commandCreate := &discordgo.ApplicationCommand{
		Name:        cmd.Name,
		Description: cmd.Description,
		Options:     options,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := h.session.ApplicationCommandCreate(
			h.config.Discord.ClientID,
			h.config.Discord.GuildID,
			commandCreate,
		)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("command registration timed out for %s", name)
	case err := <-done:
		if err != nil {
			return err
		}
	}

	h.commandMutex.Lock()
	h.commands[name] = cmd
	h.commandMutex.Unlock()

	return nil
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run(s, i, h.config)
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
