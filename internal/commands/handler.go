package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/kevinfinalboss/Void/commands/all"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/logger"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

type Handler struct {
	commands  map[string]*types.Command
	session   *discordgo.Session
	config    *config.Config
	logger    *logger.Logger
	cooldowns map[string]map[string]int64
}

func NewHandler(s *discordgo.Session, cfg *config.Config, l *logger.Logger) *Handler {
	return &Handler{
		commands:  make(map[string]*types.Command),
		session:   s,
		config:    cfg,
		logger:    l,
		cooldowns: make(map[string]map[string]int64),
	}
}

func (h *Handler) LoadCommands() error {
	h.logger.Info("Loading commands...")

	if err := h.DeleteCommands(); err != nil {
		h.logger.Error(fmt.Sprintf("Error deleting existing commands: %v", err))
	}

	for name, cmd := range registry.Commands {
		if err := h.registerCommand(name, cmd); err != nil {
			h.logger.Error(fmt.Sprintf("Error registering command %s: %v", name, err))
			continue
		}
	}

	h.logger.Info(fmt.Sprintf("Loaded %d commands successfully", len(h.commands)))
	return nil
}

func (h *Handler) registerCommand(name string, cmd *types.Command) error {
	var options []*discordgo.ApplicationCommandOption

	for _, opt := range cmd.Options {
		discordOpt := &discordgo.ApplicationCommandOption{
			Name:        opt.Name,
			Description: opt.Description,
			Type:        opt.Type,
			Required:    opt.Required,
			Choices:     opt.Choices,
		}
		options = append(options, discordOpt)
	}

	_, err := h.session.ApplicationCommandCreate(
		h.config.Discord.ClientID,
		h.config.Discord.GuildID,
		&discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     options,
		},
	)

	if err != nil {
		return fmt.Errorf("error registering command %s: %v", name, err)
	}

	h.commands[name] = cmd
	h.logger.Info(fmt.Sprintf("Registered command: %s", name))
	return nil
}

func (h *Handler) DeleteCommands() error {
	commands, err := h.session.ApplicationCommands(h.config.Discord.ClientID, h.config.Discord.GuildID)
	if err != nil {
		return fmt.Errorf("error fetching existing commands: %v", err)
	}

	for _, cmd := range commands {
		err := h.session.ApplicationCommandDelete(h.config.Discord.ClientID, h.config.Discord.GuildID, cmd.ID)
		if err != nil {
			h.logger.Error(fmt.Sprintf("Error deleting command %s: %v", cmd.Name, err))
		}
	}

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

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		h.logger.Error("Não foi possível identificar o usuário na interação.")
		return
	}

	commandName := i.ApplicationCommandData().Name
	cmd, exists := h.commands[commandName]
	if !exists {
		h.logger.Error(fmt.Sprintf("Command not found: %s", commandName))
		return
	}

	if !h.checkCooldown(userID, commandName) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Por favor, aguarde antes de usar este comando novamente.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := cmd.Run(s, i, h.config); err != nil {
		h.logger.Error(fmt.Sprintf("Error executing command %s: %v", commandName, err))
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Ocorreu um erro ao executar o comando.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

func (h *Handler) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commandName := i.ApplicationCommandData().Name
	cmd, exists := h.commands[commandName]
	if !exists || cmd.AutoComplete == nil {
		return
	}

	choices, err := cmd.AutoComplete(s, i)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Error in autocomplete for command %s: %v", commandName, err))
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		h.logger.Error(fmt.Sprintf("Error sending autocomplete response: %v", err))
	}
}

func (h *Handler) checkCooldown(userID, commandName string) bool {
	if h.cooldowns[commandName] == nil {
		h.cooldowns[commandName] = make(map[string]int64)
	}

	lastUsage, exists := h.cooldowns[commandName][userID]
	if !exists {
		h.cooldowns[commandName][userID] = time.Now().Unix()
		return true
	}

	if time.Now().Unix()-lastUsage < 5 {
		return false
	}

	h.cooldowns[commandName][userID] = time.Now().Unix()
	return true
}
