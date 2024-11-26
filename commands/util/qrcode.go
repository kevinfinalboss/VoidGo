package util

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
	"github.com/skip2/go-qrcode"
)

func init() {
	registry.RegisterCommand(QRCodeCommand)
}

var QRCodeCommand = &types.Command{
	Name:        "qrcode",
	Description: "Gera um QR Code a partir de um link fornecido",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "link",
			Description: "O link para gerar o QR Code",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return fmt.Errorf("falha ao responder à interação deferida: %w", err)
		}

		options := i.ApplicationCommandData().Options
		var link string
		for _, opt := range options {
			if opt.Name == "link" {
				link = opt.StringValue()
				break
			}
		}

		if link == "" {
			errorEmbed := &discordgo.MessageEmbed{
				Title:       "❌ Erro",
				Description: "Você deve fornecer um link válido.",
				Color:       0xff0000,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: &discordgo.MessageEmbedFooter{
					Text:    "Devil • Criado por KevinFinalBoss",
					IconURL: "https://github.com/kevinfinalboss.png",
				},
			}

			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{errorEmbed},
			})
			if err != nil {
				return fmt.Errorf("falha ao editar a resposta com o embed de erro: %w", err)
			}

			return nil
		}

		qr, err := qrcode.Encode(link, qrcode.Medium, 256)
		if err != nil {
			errorEmbed := &discordgo.MessageEmbed{
				Title:       "❌ Erro",
				Description: "Ocorreu um erro ao gerar o QR Code.",
				Color:       0xff0000,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: &discordgo.MessageEmbedFooter{
					Text:    "Devil • Criado por KevinFinalBoss",
					IconURL: "https://github.com/kevinfinalboss.png",
				},
			}

			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{errorEmbed},
			})
			if err != nil {
				return fmt.Errorf("falha ao editar a resposta com o embed de erro: %w", err)
			}

			return nil
		}

		file := &discordgo.File{
			Name:        "qrcode.png",
			ContentType: "image/png",
			Reader:      bytes.NewReader(qr),
		}

		successEmbed := &discordgo.MessageEmbed{
			Title:       "📷 Seu QR Code",
			Description: "Aqui está o seu QR Code:",
			Color:       0x00ff00,
			Timestamp:   time.Now().Format(time.RFC3339),
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://qrcode.png",
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "Devil • Criado por KevinFinalBoss",
				IconURL: "https://github.com/kevinfinalboss.png",
			},
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{successEmbed},
			Files:  []*discordgo.File{file},
		})
		if err != nil {
			return fmt.Errorf("falha ao editar a resposta com o QR Code: %w", err)
		}

		return nil
	},
}
