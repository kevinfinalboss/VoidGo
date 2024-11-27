package lol

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/api/services"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

var (
	assetsPath = "assets/champions/icons"
	baseURL    = "https://voidgo.kevindev.com.br/assets/champions/icons"
)

func init() {
	registry.RegisterCommand(ChampionRotationCommand)
}

const MaxFieldLength = 1024

var ChampionRotationCommand = &types.Command{
	Name:        "champion-rotation",
	Description: "Mostra a rotação gratuita de campeões do League of Legends",
	Category:    "League of Legends",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "region",
			Description: "Região do servidor",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "Brasil", Value: "br1"},
				{Name: "Norte América", Value: "na1"},
				{Name: "Europa Oeste", Value: "euw1"},
				{Name: "Europa Nórdica & Leste", Value: "eun1"},
				{Name: "Coreia", Value: "kr"},
			},
		},
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return fmt.Errorf("erro ao enviar resposta inicial: %v", err)
		}

		options := i.ApplicationCommandData().Options
		region := options[0].StringValue()

		riotService := services.NewRiotService(cfg.Riot.APIKey)

		rotations, err := riotService.GetChampionRotations(region)
		if err != nil {
			return fmt.Errorf("erro ao obter rotações: %v", err)
		}

		champData, err := riotService.GetChampionsData()
		if err != nil {
			return fmt.Errorf("erro ao obter dados dos campeões: %v", err)
		}

		var fields []*discordgo.MessageEmbedField

		freeChampImages := getChampionImagesLocal(rotations.FreeChampionIds, champData)
		chunks := chunkImageUrls(freeChampImages, MaxFieldLength)
		for i, chunk := range chunks {
			name := "🎮 Campeões Gratuitos da Semana"
			if i > 0 {
				name = "\u200B"
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   name,
				Value:  chunk,
				Inline: false,
			})
		}

		newPlayerImages := getChampionImagesLocal(rotations.FreeChampionIdsForNewPlayers, champData)
		chunks = chunkImageUrls(newPlayerImages, MaxFieldLength)
		for i, chunk := range chunks {
			name := "🌟 Campeões Gratuitos para Novos Jogadores"
			if i > 0 {
				name = "\u200B"
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   name,
				Value:  chunk,
				Inline: false,
			})
		}

		embed := &discordgo.MessageEmbed{
			Title:  "Rotação Gratuita de Campeões",
			Color:  0x0099FF,
			Fields: fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    fmt.Sprintf("Região: %s • Devil", region),
				IconURL: "https://github.com/kevinfinalboss.png",
			},
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf("%s/Devil.png", baseURL),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}

func getChampionImagesLocal(ids []int, champData *services.ChampionData) []string {
	var images []string
	for _, id := range ids {
		for _, champ := range champData.Data {
			if champ.Key == fmt.Sprintf("%d", id) {
				imageUrl := fmt.Sprintf("%s/%s.png", baseURL, champ.Image.Full[:len(champ.Image.Full)-4])
				images = append(images, imageUrl)
				break
			}
		}
	}
	return images
}

func chunkImageUrls(urls []string, maxLength int) []string {
	var chunks []string
	currentChunk := ""

	for _, url := range urls {
		if len(currentChunk)+len(url) > maxLength {
			chunks = append(chunks, currentChunk)
			currentChunk = url
		} else {
			if currentChunk != "" {
				currentChunk += " "
			}
			currentChunk += url
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
