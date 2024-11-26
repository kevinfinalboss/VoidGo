package images

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(ResizeImageCommand)
}

var ResizeImageCommand = &types.Command{
	Name:        "resize-image",
	Description: "Redimensiona uma imagem para a largura e altura especificadas",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "imagem",
			Description: "A imagem para redimensionar",
			Type:        discordgo.ApplicationCommandOptionAttachment,
			Required:    true,
		},
		{
			Name:        "largura",
			Description: "A nova largura da imagem",
			Type:        discordgo.ApplicationCommandOptionInteger,
			Required:    true,
		},
		{
			Name:        "altura",
			Description: "A nova altura da imagem",
			Type:        discordgo.ApplicationCommandOptionInteger,
			Required:    true,
		},
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{},
		})
		if err != nil {
			return fmt.Errorf("falha ao enviar a resposta inicial: %v", err)
		}

		options := i.ApplicationCommandData().Options
		if len(options) < 3 {
			return respondWithError(s, i, "Parâmetros insuficientes.")
		}

		attachment := i.ApplicationCommandData().Resolved.Attachments[options[0].Value.(string)]
		if attachment == nil {
			return respondWithError(s, i, "Falha ao resolver o anexo.")
		}

		width := options[1].IntValue()
		height := options[2].IntValue()

		if width <= 0 || height <= 0 {
			return respondWithError(s, i, "Largura e altura devem ser maiores que zero.")
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

		transformation := fmt.Sprintf("c_scale,w_%d,h_%d", width, height)

		uploadResult, err := cld.Upload.Upload(ctx, bytes.NewReader(imageData), uploader.UploadParams{
			Transformation: transformation,
		})
		if err != nil {
			return respondWithError(s, i, "Erro ao fazer upload da imagem para o Cloudinary.")
		}

		resizedImageData, err := downloadFile(uploadResult.SecureURL)
		if err != nil {
			return respondWithError(s, i, "Falha ao baixar a imagem redimensionada.")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Imagem Redimensionada",
			Color:       0x00ff00,
			Description: "✅ Redimensionamento concluído com sucesso!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Nova Largura",
					Value:  fmt.Sprintf("%d px", width),
					Inline: true,
				},
				{
					Name:   "Nova Altura",
					Value:  fmt.Sprintf("%d px", height),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Redimensionador de Imagens",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
			Files: []*discordgo.File{
				{
					Name:   "redimensionada.png",
					Reader: bytes.NewReader(resizedImageData),
				},
			},
		})

		return err
	},
}
