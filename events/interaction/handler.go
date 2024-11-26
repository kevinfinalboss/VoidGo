package interaction

import (
	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/logger"
	"github.com/kevinfinalboss/Void/internal/types"
)

type Handler struct {
	session  *discordgo.Session
	logger   *logger.Logger
	commands map[string]*types.Command
	config   *config.Config
}

func NewHandler(s *discordgo.Session, l *logger.Logger, cmds map[string]*types.Command, cfg *config.Config) *Handler {
	return &Handler{
		session:  s,
		logger:   l,
		commands: cmds,
		config:   cfg,
	}
}

func (h *Handler) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		h.handleCommand(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		h.handleAutoComplete(s, i)
	}
}

func (h *Handler) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commandName := i.ApplicationCommandData().Name
	if cmd, ok := h.commands[commandName]; ok {
		if err := cmd.Run(s, i, h.config); err != nil {
			h.logger.Error("Error executing command:", err)
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Ocorreu um erro ao executar o comando.",
				},
			})
			if err != nil {
				h.logger.Error("Error responding to interaction:", err)
			}
		}
	}
}

func (h *Handler) handleAutoComplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdName := i.ApplicationCommandData().Name
	if cmd, ok := h.commands[cmdName]; ok && cmd.AutoComplete != nil {
		choices, err := cmd.AutoComplete(s, i)
		if err != nil {
			h.logger.Error("Error in autocomplete:", err)
			return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})
		if err != nil {
			h.logger.Error("Error responding to autocomplete:", err)
		}
	}
}
