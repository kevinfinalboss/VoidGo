package util

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(UptimeCommand)
}

var UptimeCommand = &types.Command{
	Name:        "uptime",
	Description: "Exibe o tempo de atividade do bot",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		uptime := time.Since(cfg.BotStartTime)

		days := int(uptime.Hours()) / 24
		hours := int(uptime.Hours()) % 24
		minutes := int(uptime.Minutes()) % 60
		seconds := int(uptime.Seconds()) % 60

		uptimeStr := fmt.Sprintf("%d dias, %d horas, %d minutos, %d segundos", days, hours, minutes, seconds)

		embed := &discordgo.MessageEmbed{
			Title:       "⏱️ Tempo de Atividade",
			Description: fmt.Sprintf("O bot está online há:\n**%s**", uptimeStr),
			Color:       0x00ff00,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Iniciado em",
					Value:  cfg.BotStartTime.Format("02/01/2006 15:04:05"),
					Inline: true,
				},
				{
					Name:   "Atualização",
					Value:  time.Now().Format("02/01/2006 15:04:05"),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "Devil • Tempo de Atividade",
				IconURL: s.State.User.AvatarURL(""),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
	},
}
