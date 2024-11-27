package util

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
	"github.com/nfnt/resize"
)

func init() {
	registry.RegisterCommand(AddEmojiCommand)
}

var AddEmojiCommand = &types.Command{
	Name:        "addemoji",
	Description: "Adiciona uma imagem como emoji no servidor",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	AdminOnly:   true,
	Options: []*types.CommandOption{
		{
			Name:        "nome",
			Description: "Nome para o emoji",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
		{
			Name:        "imagem",
			Description: "Imagem para usar como emoji (suporta PNG, JPG, GIF e WEBP)",
			Type:        discordgo.ApplicationCommandOptionAttachment,
			Required:    true,
		},
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		// Resposta inicial diferida
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return err
		}

		// Verifica permissões do usuário
		member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
		if err != nil {
			return respondWithError(s, i, "Erro ao verificar permissões do usuário")
		}

		hasPermission := false
		for _, roleID := range member.Roles {
			role, err := s.State.Role(i.GuildID, roleID)
			if err != nil {
				continue
			}
			if role.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return respondWithError(s, i, "Você precisa ser administrador para usar este comando")
		}

		// Obtém os dados do comando
		data := i.ApplicationCommandData()
		emojiName := strings.ToLower(data.Options[0].StringValue())

		// Obtém o attachment
		var attachmentID string
		for _, option := range data.Options {
			if option.Name == "imagem" {
				attachmentID = option.Value.(string)
				break
			}
		}

		attachment := data.Resolved.Attachments[attachmentID]

		// Verifica se o nome do emoji é válido
		if len(emojiName) < 2 || len(emojiName) > 32 {
			return respondWithError(s, i, "O nome do emoji deve ter entre 2 e 32 caracteres")
		}

		// Verifica se a imagem é muito grande
		var maxSize int64
		if isAnimated(attachment.Filename) {
			maxSize = 512000 // 512KB para emojis animados
		} else {
			maxSize = 256000 // 256KB para emojis estáticos
		}

		if int64(attachment.Size) > maxSize {
			return respondWithError(s, i, fmt.Sprintf("A imagem é muito grande. O limite é %dKB", maxSize/1000))
		}

		// Baixa a imagem
		resp, err := http.Get(attachment.URL)
		if err != nil {
			return respondWithError(s, i, "Erro ao baixar a imagem")
		}
		defer resp.Body.Close()

		// Lê todos os bytes da imagem
		imageBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return respondWithError(s, i, "Erro ao ler a imagem")
		}

		// Se for uma imagem estática, redimensiona
		if !isAnimated(attachment.Filename) {
			img, format, err := image.Decode(bytes.NewReader(imageBytes))
			if err != nil {
				return respondWithError(s, i, "Formato de imagem inválido")
			}

			// Redimensiona a imagem para 128x128
			resizedImg := resize.Resize(128, 128, img, resize.Lanczos3)

			var buf bytes.Buffer
			switch format {
			case "jpeg", "jpg":
				err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 90})
			case "png":
				err = png.Encode(&buf, resizedImg)
			default:
				return respondWithError(s, i, "Formato de imagem não suportado para imagens estáticas. Use PNG ou JPG")
			}
			if err != nil {
				return respondWithError(s, i, "Erro ao processar a imagem")
			}
			imageBytes = buf.Bytes()
		}

		// Cria o emoji
		emoji, err := s.GuildEmojiCreate(i.GuildID, &discordgo.EmojiParams{
			Name:  emojiName,
			Image: fmt.Sprintf("data:image/%s;base64,%s", getImageFormat(attachment.Filename), base64.StdEncoding.EncodeToString(imageBytes)),
		})
		if err != nil {
			return respondWithError(s, i, "Erro ao criar o emoji. Verifique se o bot tem permissões adequadas e se há espaço disponível para novos emojis")
		}

		// Responde com sucesso
		embed := &discordgo.MessageEmbed{
			Title: "✅ Emoji Adicionado!",
			Description: fmt.Sprintf("O emoji `:%s:` foi adicionado com sucesso!\nTipo: %s",
				emoji.Name,
				getEmojiType(attachment.Filename)),
			Color: 0x00ff00,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: attachment.URL,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}

func isAnimated(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".gif" || ext == ".webp"
}

func getImageFormat(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".gif":
		return "gif"
	case ".webp":
		return "webp"
	case ".jpg", ".jpeg":
		return "jpeg"
	default:
		return "png"
	}
}

func getEmojiType(filename string) string {
	if isAnimated(filename) {
		return "Emoji Animado"
	}
	return "Emoji Estático"
}
