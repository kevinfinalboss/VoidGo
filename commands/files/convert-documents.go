package files

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(ConvertDocumentCommand)
}

var ConvertDocumentCommand = &types.Command{
	Name:        "convert-document",
	Description: "Converte documentos entre PDF e DOCX",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "arquivo",
			Description: "O arquivo PDF ou DOCX para converter",
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

		var sourceFormat, targetFormat string

		if strings.HasSuffix(strings.ToLower(attachment.Filename), ".pdf") {
			sourceFormat = "pdf"
			targetFormat = "docx"
		} else if strings.HasSuffix(strings.ToLower(attachment.Filename), ".docx") {
			sourceFormat = "docx"
			targetFormat = "pdf"
		} else {
			return respondWithError(s, i, "Por favor, forneça um arquivo PDF ou DOCX válido.")
		}

		fileData, err := downloadFile(attachment.URL)
		if err != nil {
			return respondWithError(s, i, "Falha ao baixar o arquivo.")
		}

		fileBuffer := bytes.NewReader(fileData)

		convertedData, err := convertDocument(cfg.ConvertAPI.Secret, fileBuffer, sourceFormat, targetFormat)
		if err != nil {
			return respondWithError(s, i, fmt.Sprintf("Erro ao converter o arquivo: %v", err))
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Conversão de Documento",
			Color:       0x00ff00,
			Description: fmt.Sprintf("✅ Conversão de `%s` para `%s` concluída com sucesso!", strings.ToUpper(sourceFormat), strings.ToUpper(targetFormat)),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Arquivo Original",
					Value:  fmt.Sprintf("`%s`", attachment.Filename),
					Inline: true,
				},
				{
					Name:   "Tamanho Original",
					Value:  fmt.Sprintf("%.2f KB", float64(len(fileData))/1024),
					Inline: true,
				},
				{
					Name:   "Tamanho Convertido",
					Value:  fmt.Sprintf("%.2f KB", float64(len(convertedData))/1024),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Void Bot • Conversão de Documentos",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		convertedFileName := fmt.Sprintf("convertido.%s", targetFormat)

		var userID string
		if i.Interaction.User != nil {
			userID = i.Interaction.User.ID
		} else if i.Interaction.Member != nil && i.Interaction.Member.User != nil {
			userID = i.Interaction.Member.User.ID
		} else {
			return respondWithError(s, i, "Não foi possível identificar o usuário.")
		}

		var channelID string

		if i.GuildID == "" {
			channelID = i.ChannelID
		} else {
			dmChannel, err := s.UserChannelCreate(userID)
			if err != nil {
				return respondWithError(s, i, "Não foi possível enviar mensagem direta para você. Verifique se suas DMs estão abertas.")
			}
			channelID = dmChannel.ID

			_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Embeds: &[]*discordgo.MessageEmbed{
					{
						Title:       "Conversão de Documento",
						Color:       0x00ff00,
						Description: "✅ Seu arquivo convertido foi enviado por mensagem direta.",
						Footer: &discordgo.MessageEmbedFooter{
							Text: "Void Bot • Conversão de Documentos",
						},
						Timestamp: time.Now().Format(time.RFC3339),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("falha ao editar a resposta da interação: %v", err)
			}
		}

		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
			Files: []*discordgo.File{
				{
					Name:   convertedFileName,
					Reader: bytes.NewReader(convertedData),
				},
			},
		})
		if err != nil {
			return fmt.Errorf("falha ao enviar o arquivo convertido: %v", err)
		}

		return nil
	},
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func convertDocument(secret string, file io.Reader, sourceFormat, targetFormat string) ([]byte, error) {
	apiURL := fmt.Sprintf("https://v2.convertapi.com/convert/%s/to/%s", sourceFormat, targetFormat)

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("File", "input."+sourceFormat)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	writer.Close()

	req, err := http.NewRequest("POST", apiURL, &requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+secret)

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]interface{}
		json.Unmarshal(bodyBytes, &errorResponse)
		errorMessage := fmt.Sprintf("Status Code: %d, Message: %v", resp.StatusCode, errorResponse["Message"])
		return nil, fmt.Errorf(errorMessage)
	}

	var jsonResponse map[string]interface{}
	err = json.Unmarshal(bodyBytes, &jsonResponse)
	if err != nil {
		return nil, err
	}

	if msg, exists := jsonResponse["Message"]; exists {
		return nil, fmt.Errorf("erro na conversão: %v", msg)
	}

	files, ok := jsonResponse["Files"].([]interface{})
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("nenhum arquivo retornado pela API")
	}

	fileInfo, ok := files[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("formato de resposta inválido")
	}

	fileDataBase64, ok := fileInfo["FileData"].(string)
	if !ok {
		return nil, fmt.Errorf("dados do arquivo convertido não encontrados")
	}

	convertedData, err := base64.StdEncoding.DecodeString(fileDataBase64)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar o arquivo convertido: %v", err)
	}

	return convertedData, nil
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Erro",
		Description: message,
		Color:       0xff0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Devil • Conversão de Documentos",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	return err
}
