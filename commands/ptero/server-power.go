package ptero

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/models"
	"github.com/kevinfinalboss/Void/internal/ptero"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

var globalConfig *config.Config

func SetConfig(cfg *config.Config) {
	globalConfig = cfg
}

func init() {
	registry.RegisterCommand(ServerPowerCommand)
}

var ServerPowerCommand = &types.Command{
	Name:        "server-power",
	Description: "Gerenciar energia do servidor (ligar/desligar/reiniciar)",
	Category:    "Pterodactyl",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "servidor",
			Description: "Selecione o servidor",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
		{
			Name:        "acao",
			Description: "Ação a ser executada",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Ligar",
					Value: "start",
				},
				{
					Name:  "Desligar",
					Value: "stop",
				},
				{
					Name:  "Reiniciar",
					Value: "restart",
				},
			},
		},
	},
	AutoComplete: func(s *discordgo.Session, i *discordgo.InteractionCreate) ([]*discordgo.ApplicationCommandOptionChoice, error) {
		if globalConfig == nil {
			return nil, fmt.Errorf("configuração não inicializada")
		}

		data := i.ApplicationCommandData()
		var focusedOption *discordgo.ApplicationCommandInteractionDataOption

		for _, opt := range data.Options {
			if opt.Focused {
				focusedOption = opt
				break
			}
		}

		if focusedOption == nil || focusedOption.Name != "servidor" {
			return nil, nil
		}

		fmt.Printf("🔍 Buscando servidores para termo: %s\n", focusedOption.StringValue())

		// Criar cliente com a configuração global
		client := ptero.NewPteroClient(globalConfig)
		resp, err := client.ListServers()
		if err != nil {
			fmt.Printf("❌ Erro ao buscar servidores: %v\n", err)
			return nil, err
		}

		var serverList models.ServerListResponse
		if err := json.Unmarshal(resp, &serverList); err != nil {
			fmt.Printf("❌ Erro ao decodificar resposta: %v\n", err)
			return nil, err
		}

		var choices []*discordgo.ApplicationCommandOptionChoice
		searchTerm := strings.ToLower(focusedOption.StringValue())

		for _, server := range serverList.Data {
			serverName := server.Attributes.Name
			if searchTerm == "" || strings.Contains(strings.ToLower(serverName), searchTerm) {
				choice := &discordgo.ApplicationCommandOptionChoice{
					Name:  serverName,
					Value: server.Attributes.Identifier,
				}
				choices = append(choices, choice)
				fmt.Printf("✅ Adicionado servidor: %s\n", serverName)
			}

			if len(choices) >= 25 {
				break
			}
		}

		fmt.Printf("📋 Total de opções encontradas: %d\n", len(choices))
		return choices, nil
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return err
		}

		options := i.ApplicationCommandData().Options
		var serverID, action string

		for _, opt := range options {
			switch opt.Name {
			case "servidor":
				serverID = opt.StringValue()
			case "acao":
				action = opt.StringValue()
			}
		}

		client := ptero.NewPteroClient(cfg)
		err = client.SendPowerAction(serverID, action)
		if err != nil {
			return respondWithError(s, i, "Erro ao executar ação no servidor!")
		}

		var actionMsg string
		switch action {
		case "start":
			actionMsg = "ligado"
		case "stop":
			actionMsg = "desligado"
		case "restart":
			actionMsg = "reiniciado"
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🔧 Gerenciamento de Energia",
			Description: fmt.Sprintf("O servidor está sendo %s...", actionMsg),
			Color:       0x00ff00,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💻 Powered by Pterodactyl API",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}
