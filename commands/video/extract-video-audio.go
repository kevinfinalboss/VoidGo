package video

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(ExtractAudioCommand)
}

var ExtractAudioCommand = &types.Command{
	Name:        "extract-audio",
	Description: "Extrai o áudio de um vídeo em formato MP3",
	Category:    "Utilidade",
	Cooldown:    120 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "video",
			Description: "O vídeo para extrair o áudio",
			Type:        discordgo.ApplicationCommandOptionAttachment,
			Required:    true,
		},
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return fmt.Errorf("falha ao enviar a resposta inicial: %v", err)
		}

		options := i.ApplicationCommandData().Options
		if len(options) == 0 {
			return respondWithError(s, i, "Nenhum arquivo foi fornecido.")
		}

		attachment := i.ApplicationCommandData().Resolved.Attachments[options[0].Value.(string)]
		if attachment == nil {
			return respondWithError(s, i, "Falha ao resolver o anexo.")
		}

		videoData, err := downloadFile(attachment.URL)
		if err != nil {
			return respondWithError(s, i, "Falha ao baixar o vídeo.")
		}

		cld, err := cloudinary.NewFromParams(
			cfg.Cloudinary.CloudName,
			cfg.Cloudinary.APIKey,
			cfg.Cloudinary.APISecret,
		)
		if err != nil {
			return respondWithError(s, i, "Erro ao configurar o Cloudinary.")
		}

		ctx := context.Background()

		uploadResult, err := cld.Upload.Upload(ctx, bytes.NewReader(videoData), uploader.UploadParams{
			ResourceType:   "video",
			Format:         "mp3",
			Transformation: "f_mp3,ac_none",
		})
		if err != nil {
			return respondWithError(s, i, "Erro ao extrair o áudio do vídeo.")
		}

		fileName := strings.TrimSuffix(attachment.Filename, filepath.Ext(attachment.Filename))

		embed := &discordgo.MessageEmbed{
			Title:       "🎵 Extração de Áudio Concluída",
			Color:       0x00ff00,
			Description: "O áudio do seu vídeo foi extraído com sucesso!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "🔊 Download do Áudio",
					Value:  fmt.Sprintf("[%s.mp3](%s)", fileName, uploadResult.SecureURL),
					Inline: false,
				},
				{
					Name:   "📁 Arquivo Original",
					Value:  fmt.Sprintf("`%s`", attachment.Filename),
					Inline: true,
				},
				{
					Name:   "💿 Formato de Saída",
					Value:  "`MP3`",
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Extrator de Áudio",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		var userID string
		if i.Interaction.User != nil {
			userID = i.Interaction.User.ID
		} else if i.Interaction.Member != nil && i.Interaction.Member.User != nil {
			userID = i.Interaction.Member.User.ID
		} else {
			return respondWithError(s, i, "Não foi possível identificar o usuário.")
		}

		dmChannel, err := s.UserChannelCreate(userID)
		if err != nil {
			return respondWithError(s, i, "Não foi possível enviar mensagem direta para você. Verifique se suas DMs estão abertas.")
		}

		_, err = s.ChannelMessageSendEmbed(dmChannel.ID, embed)
		if err != nil {
			return fmt.Errorf("falha ao enviar mensagem direta: %v", err)
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "✅ Extração Concluída",
					Description: "O áudio extraído foi enviado por mensagem direta!",
					Color:       0x00ff00,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Devil • Extrator de Áudio",
					},
				},
			},
		})

		return err
	},
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Erro",
		Description: message,
		Color:       0xFF0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Devil • Extrator de Áudio",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	return err
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
