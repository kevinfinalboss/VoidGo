package images

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(ImageInfoCommand)
}

var ImageInfoCommand = &types.Command{
	Name:        "image-info",
	Description: "Obtém informações detalhadas sobre uma imagem enviada",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "imagem",
			Description: "A imagem para obter informações",
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

		imageData, err := downloadFile(attachment.URL)
		if err != nil {
			return respondWithError(s, i, "Falha ao baixar a imagem.")
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

		uploadResult, err := cld.Upload.Upload(ctx, bytes.NewReader(imageData), uploader.UploadParams{})
		if err != nil {
			return respondWithError(s, i, "Erro ao fazer upload da imagem para o Cloudinary.")
		}

		imageInfo, err := cld.Admin.Asset(ctx, admin.AssetParams{PublicID: uploadResult.PublicID})
		if err != nil {
			return respondWithError(s, i, "Erro ao obter informações da imagem.")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "📷 Informações da Imagem",
			Color:       0x00ff00,
			Description: "Aqui estão as informações detalhadas da imagem enviada:",
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: uploadResult.SecureURL,
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Formato",
					Value:  fmt.Sprintf("`%s`", imageInfo.Format),
					Inline: true,
				},
				{
					Name:   "Dimensões",
					Value:  fmt.Sprintf("`%dx%d px`", imageInfo.Width, imageInfo.Height),
					Inline: true,
				},
				{
					Name:   "Tamanho do Arquivo",
					Value:  fmt.Sprintf("`%.2f KB`", float64(imageInfo.Bytes)/1024),
					Inline: true,
				},
				{
					Name:   "Tipo de Recurso",
					Value:  fmt.Sprintf("`%s`", imageInfo.ResourceType),
					Inline: true,
				},
				{
					Name:   "Criado em",
					Value:  fmt.Sprintf("`%s`", imageInfo.CreatedAt.Format("02/01/2006 15:04:05")),
					Inline: true,
				},
				{
					Name:   "URL",
					Value:  fmt.Sprintf("[Clique aqui](%s)", uploadResult.SecureURL),
					Inline: false,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Informações da Imagem",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}
