package admin

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/database"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

var (
	db            *database.MongoDB
	awaitingAudit sync.Map
)

func SetDatabase(mongodb *database.MongoDB) {
	db = mongodb
}

func init() {
	registry.RegisterCommand(ConfigCommand)
}

var ConfigCommand = &types.Command{
	Name:        "config",
	Description: "Configure as opções do servidor",
	Category:    "Administração",
	AdminOnly:   true,
	Cooldown:    5 * time.Second,
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		embed := &discordgo.MessageEmbed{
			Title:       "⚙️ Configurações do Servidor",
			Description: "Selecione uma opção abaixo para configurar:",
			Color:       0x2B2D31,
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Configurações",
			},
		}

		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Definir Canal de Audit",
						Style:    discordgo.PrimaryButton,
						CustomID: "btn_set_audit",
						Emoji: &discordgo.ComponentEmoji{
							Name: "📝",
						},
					},
				},
			},
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	},
}

func HandleConfigButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	switch i.MessageComponentData().CustomID {
	case "btn_set_audit":
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "Por favor, mencione o canal que deseja definir como canal de audit (#canal):",
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("Erro ao responder à interação do botão: %v\n", err)
			return
		}

		userID := i.Member.User.ID
		channelID := i.ChannelID
		guildID := i.GuildID

		awaitingAudit.Store(userID, true)

		handler := func(s *discordgo.Session, m *discordgo.MessageCreate) {
			if m.Author.ID != userID || m.ChannelID != channelID {
				return
			}

			if _, ok := awaitingAudit.Load(userID); !ok {
				return
			}

			awaitingAudit.Delete(userID)

			re := regexp.MustCompile(`<#(\d+)>`)
			matches := re.FindStringSubmatch(m.Content)
			if len(matches) < 2 {
				s.ChannelMessageSend(channelID, "❌ Formato inválido. Por favor, mencione o canal usando #.")
				return
			}

			targetChannelID := matches[1]

			targetChannel, err := s.State.Channel(targetChannelID)
			if err != nil || targetChannel.GuildID != guildID {
				targetChannel, err = s.Channel(targetChannelID)
				if err != nil || targetChannel.GuildID != guildID {
					s.ChannelMessageSend(channelID, "❌ Canal não encontrado ou não pertence a este servidor. Por favor, mencione um canal válido usando #.")
					return
				}
			}

			if err := db.UpdateGuildSettings(guildID, "audit_log_channel", targetChannel.ID); err != nil {
				s.ChannelMessageSend(channelID, "❌ Erro ao salvar o canal de audit.")
				return
			}

			embed := &discordgo.MessageEmbed{
				Title:       "✅ Configuração Salva",
				Description: fmt.Sprintf("Canal de audit definido como <#%s>", targetChannel.ID),
				Color:       0x00FF00,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Devil • Configurações",
				},
			}

			s.ChannelMessageSendEmbed(channelID, embed)
		}

		removeHandler := s.AddHandler(handler)

		time.AfterFunc(time.Minute, func() {
			removeHandler()
			if _, ok := awaitingAudit.Load(userID); ok {
				awaitingAudit.Delete(userID)
				s.ChannelMessageSend(channelID, "❌ Tempo esgotado. Por favor, tente novamente.")
			}
		})
	}
}
