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
	registry.RegisterCommand(BlurImageCommand)
}

var BlurImageCommand = &types.Command{
	Name:        "blur-image",
	Description: "Aplica um efeito de desfoque na imagem",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "imagem",
			Description: "A imagem para aplicar o desfoque",
			Type:        discordgo.ApplicationCommandOptionAttachment,
			Required:    true,
		},
		{
			Name:        "intensidade",
			Description: "Intensidade do desfoque (1-200)",
			Type:        discordgo.ApplicationCommandOptionInteger,
			Required:    false,
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
		if len(options) == 0 {
			return respondWithError(s, i, "Parâmetros insuficientes.")
		}

		attachment := i.ApplicationCommandData().Resolved.Attachments[options[0].Value.(string)]
		if attachment == nil {
			return respondWithError(s, i, "Falha ao resolver o anexo.")
		}

		intensity := int64(50)
		if len(options) > 1 {
			intensity = options[1].IntValue()
			if intensity < 1 || intensity > 200 {
				return respondWithError(s, i, "Intensidade inválida. Use um valor entre 1 e 200.")
			}
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

		transformation := fmt.Sprintf("e_blur:%d", intensity)

		uploadResult, err := cld.Upload.Upload(ctx, bytes.NewReader(imageData), uploader.UploadParams{
			Transformation: transformation,
		})
		if err != nil {
			return respondWithError(s, i, "Erro ao fazer upload da imagem para o Cloudinary.")
		}

		blurredImageData, err := downloadFile(uploadResult.SecureURL)
		if err != nil {
			return respondWithError(s, i, "Falha ao baixar a imagem com desfoque.")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Imagem com Desfoque",
			Color:       0x00ff00,
			Description: "✅ Desfoque aplicado com sucesso!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Intensidade",
					Value:  fmt.Sprintf("%d", intensity),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Desfoque de Imagens",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
			Files: []*discordgo.File{
				{
					Name:   "desfoque.png",
					Reader: bytes.NewReader(blurredImageData),
				},
			},
		})

		return err
	},
}
