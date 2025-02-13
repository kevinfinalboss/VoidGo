package ptero

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/models"
	"github.com/kevinfinalboss/Void/internal/ptero"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(ListServersCommand)
}

var ListServersCommand = &types.Command{
	Name:        "list-servers",
	Description: "Lista todos os servidores do Pterodactyl",
	Category:    "Pterodactyl",
	Cooldown:    5 * time.Second,
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return err
		}

		client := ptero.NewPteroClient(cfg)
		resp, err := client.ListServers()
		if err != nil {
			return respondWithError(s, i, "Erro ao buscar servidores!")
		}

		var serverList models.ServerListResponse
		err = json.Unmarshal(resp, &serverList)
		if err != nil {
			return respondWithError(s, i, "Erro ao processar a resposta do Pterodactyl.")
		}

		if len(serverList.Data) == 0 {
			return respondWithError(s, i, "🔍 Nenhum servidor encontrado.")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "📜 Lista de Servidores",
			Description: "Aqui estão os seus servidores registrados no **Pterodactyl**:",
			Color:       0x00ff00,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💻 Powered by Pterodactyl API",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		for _, server := range serverList.Data {
			status := "🔵 Online"
			if server.Attributes.State == nil {
				status = "⚪ Desconhecido"
			} else if *server.Attributes.State == "running" {
				status = "🟢 Online"
			} else if *server.Attributes.State == "offline" {
				status = "🔴 Offline"
			} else if *server.Attributes.State == "starting" {
				status = "🟡 Iniciando"
			} else if *server.Attributes.State == "stopping" {
				status = "🟠 Parando"
			}

			// Status especiais
			extraStatus := ""
			if server.Attributes.IsSuspended {
				extraStatus += "⛔ **Suspenso**\n"
			}
			if server.Attributes.IsInstalling {
				extraStatus += "⚙️ **Instalando**\n"
			}
			if server.Attributes.IsTransferring {
				extraStatus += "📤 **Transferindo**\n"
			}

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name: fmt.Sprintf("🔹 %s", server.Attributes.Name),
				Value: fmt.Sprintf(
					"**🆔 ID:** `%s`\n"+
						"**🖥️ Node:** `%s`\n"+
						"**💾 Armazenamento:** `%d MB`\n"+
						"**🧠 Memória:** `%d MB`\n"+
						"**⚙️ CPU:** `%d%%`\n"+
						"**🟢 Status:** `%s`\n"+
						"%s",
					server.Attributes.Identifier,
					server.Attributes.Node,
					server.Attributes.Disk,
					server.Attributes.Memory,
					server.Attributes.CPU,
					status,
					extraStatus,
				),
				Inline: false,
			})
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
	return err
}
