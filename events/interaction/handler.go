package interaction

import (
	"fmt"

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
			h.logger.Error(fmt.Sprintf("Erro ao executar comando: %v", err))
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Ocorreu um erro ao executar o comando.",
				},
			})
			if err != nil {
				h.logger.Error(fmt.Sprintf("Erro ao responder interação: %v", err))
			}
		}
	}
}

func (h *Handler) handleAutoComplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	cmdName := i.ApplicationCommandData().Name
	h.logger.Info(fmt.Sprintf("🔄 Autocomplete ativado para: %s", cmdName))

	cmd, ok := h.commands[cmdName]
	if !ok {
		h.logger.Error(fmt.Sprintf("Comando não encontrado: %s", cmdName))
		return
	}

	if cmd.AutoComplete == nil {
		h.logger.Error(fmt.Sprintf("Comando não tem autocomplete: %s", cmdName))
		return
	}

	choices, err := cmd.AutoComplete(s, i)
	if err != nil {
		h.logger.Error(fmt.Sprintf("Erro no autocomplete: %v", err))
		return
	}

	h.logger.Info(fmt.Sprintf("✅ Autocomplete encontrou %d opções", len(choices)))

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})

	if err != nil {
		h.logger.Error(fmt.Sprintf("Erro ao responder autocomplete: %v", err))
	}
}
