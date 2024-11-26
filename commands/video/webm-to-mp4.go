package video

import (
	"bytes"
	"context"
	"fmt"
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
	registry.RegisterCommand(WebmToMP4Command)
}

var WebmToMP4Command = &types.Command{
	Name:        "convert-webm",
	Description: "Converte um vídeo WEBM para formato MP4",
	Category:    "Utilidade",
	Cooldown:    30 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "video",
			Description: "O vídeo WEBM para converter",
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

		if !isWebmFormat(attachment.Filename) {
			return respondWithError(s, i, "Por favor, envie um arquivo no formato WEBM.")
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
			Format:         "mp4",
			Transformation: "f_mp4",
		})
		if err != nil {
			return respondWithError(s, i, "Erro ao converter o vídeo.")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🎥 Conversão de Vídeo Concluída",
			Color:       0x00ff00,
			Description: "Seu vídeo foi convertido com sucesso de WEBM para MP4!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Download",
					Value:  fmt.Sprintf("[Clique aqui para baixar](%s)", uploadResult.SecureURL),
					Inline: false,
				},
				{
					Name:   "Arquivo Original",
					Value:  fmt.Sprintf("`%s`", attachment.Filename),
					Inline: true,
				},
				{
					Name:   "Tamanho Original",
					Value:  fmt.Sprintf("`%.2f MB`", float64(attachment.Size)/(1024*1024)),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Conversor de Vídeo",
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
					Title:       "✅ Conversão Concluída",
					Description: "O resultado da conversão foi enviado por mensagem direta!",
					Color:       0x00ff00,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Devil • Conversor de Vídeo",
					},
				},
			},
		})

		return err
	},
}

func isWebmFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".webm"
}
