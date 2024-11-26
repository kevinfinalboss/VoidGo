package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(AICommand)
}

type GroqRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var AICommand = &types.Command{
	Name:        "ai",
	Description: "Gera uma resposta usando o modelo llama3-70b-8192 via Aurora - IA",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "prompt",
			Description: "O prompt para o modelo de IA",
			Type:        discordgo.ApplicationCommandOptionString,
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
			return respondWithError(s, i, "Nenhum prompt fornecido.")
		}
		prompt := options[0].StringValue()

		reqBody := GroqRequest{
			Messages: []Message{
				{
					Role:    "user",
					Content: prompt,
				},
			},
			Model: "llama3-70b-8192",
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return respondWithError(s, i, "Erro ao preparar a requisição.")
		}

		req, err := http.NewRequest("POST", "https://api.groq.com/v1/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			return respondWithError(s, i, "Erro ao criar a requisição.")
		}

		req.Header.Set("Authorization", "Bearer "+cfg.Groq.APIKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return respondWithError(s, i, "Erro ao comunicar com a API Groq.")
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return respondWithError(s, i, "Erro ao ler a resposta da API.")
		}

		var groqResponse GroqResponse
		if err := json.Unmarshal(body, &groqResponse); err != nil {
			return respondWithError(s, i, "Erro ao processar a resposta da API.")
		}

		if len(groqResponse.Choices) == 0 || groqResponse.Choices[0].Message.Content == "" {
			return respondWithError(s, i, "Não foi possível gerar uma resposta.")
		}

		var username string
		var avatarURL string
		if i.Member != nil && i.Member.User != nil {
			username = i.Member.User.Username
			avatarURL = i.Member.User.AvatarURL("")
		} else if i.User != nil {
			username = i.User.Username
			avatarURL = i.User.AvatarURL("")
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Devil - IA",
			Description: groqResponse.Choices[0].Message.Content,
			Color:       0x7289DA,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    fmt.Sprintf("Solicitado por %s", username),
				IconURL: avatarURL,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "Erro na Geração de Resposta",
		Description: message,
		Color:       0xFF0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Devil - IA • Erro",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	return err
}
