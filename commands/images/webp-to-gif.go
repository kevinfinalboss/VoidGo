package images

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(WebpToGifCommand)
}

var WebpToGifCommand = &types.Command{
	Name:        "webp-to-gif",
	Description: "Converta uma imagem WebP para o formato GIF",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "imagem",
			Description: "A imagem WebP para converter",
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

		if !isWebP(attachment.ContentType) {
			return respondWithError(s, i, "Por favor, forneça uma imagem WebP válida.")
		}

		webpData, err := downloadFile(attachment.URL)
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

		uploadResult, err := cld.Upload.Upload(ctx, bytes.NewReader(webpData), uploader.UploadParams{
			ResourceType: "image",
			Format:       "gif",
		})
		if err != nil {
			return respondWithError(s, i, "Erro ao fazer upload da imagem para o Cloudinary.")
		}

		gifURL := uploadResult.SecureURL

		gifData, err := downloadFile(gifURL)
		if err != nil {
			return respondWithError(s, i, "Falha ao baixar a imagem convertida.")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Conversão de WebP para GIF",
			Color:       0x00ff00,
			Description: "✅ Conversão concluída com sucesso!",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Tamanho Original",
					Value:  fmt.Sprintf("%.2f KB", float64(len(webpData))/1024),
					Inline: true,
				},
				{
					Name:   "Tamanho Convertido",
					Value:  fmt.Sprintf("%.2f KB", float64(len(gifData))/1024),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Devil • Conversor WebP para GIF",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
			Files: []*discordgo.File{
				{
					Name:   "convertido.gif",
					Reader: bytes.NewReader(gifData),
				},
			},
		})

		return err
	},
}

func isWebP(contentType string) bool {
	return contentType == "image/webp" || contentType == "application/octet-stream"
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Erro",
		Description: message,
		Color:       0xff0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Devil • Conversor WebP para GIF",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	return err
}
