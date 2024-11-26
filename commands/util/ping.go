package util

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

func init() {
	registry.RegisterCommand(PingCommand)
}

var PingCommand = &types.Command{
	Name:        "ping",
	Description: "Responde com informações de latência e status do bot",
	Category:    "Utilidade",
	Cooldown:    5 * time.Second,
	Options: []*types.CommandOption{
		{
			Name:        "tipo",
			Description: "Tipo de informação para exibir (basico, detalhado, sistema)",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Básico",
					Value: "basico",
				},
				{
					Name:  "Detalhado",
					Value: "detalhado",
				},
				{
					Name:  "Sistema",
					Value: "sistema",
				},
			},
		},
	},
	Run: func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error {
		start := time.Now()

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return err
		}

		options := i.ApplicationCommandData().Options
		checkType := "basico"
		if len(options) > 0 && options[0].Name == "tipo" {
			checkType = options[0].StringValue()
		}

		latency := s.HeartbeatLatency()
		restLatency := time.Since(start)

		embed := &discordgo.MessageEmbed{
			Title:     "🏓 Pong!",
			Color:     0x00ff00,
			Timestamp: time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text:    "Devil • Criado por KevinFinalBoss",
				IconURL: "https://github.com/kevinfinalboss.png",
			},
		}

		switch checkType {
		case "basico":
			embed.Fields = []*discordgo.MessageEmbedField{
				{
					Name:   "Latência do Gateway",
					Value:  fmt.Sprintf("`%dms`", latency.Milliseconds()),
					Inline: true,
				},
				{
					Name:   "Latência do REST",
					Value:  fmt.Sprintf("`%dms`", restLatency.Milliseconds()),
					Inline: true,
				},
			}

		case "detalhado":
			uptime := time.Since(cfg.BotStartTime)
			embed.Fields = []*discordgo.MessageEmbedField{
				{
					Name:   "Latência do Gateway",
					Value:  fmt.Sprintf("`%dms`", latency.Milliseconds()),
					Inline: true,
				},
				{
					Name:   "Latência do REST",
					Value:  fmt.Sprintf("`%dms`", restLatency.Milliseconds()),
					Inline: true,
				},
				{
					Name:   "Tempo de Atividade",
					Value:  fmt.Sprintf("`%s`", uptime.Round(time.Second).String()),
					Inline: true,
				},
				{
					Name:   "Shard ID",
					Value:  fmt.Sprintf("`%d`", s.ShardID),
					Inline: true,
				},
				{
					Name:   "Guildas Conectadas",
					Value:  fmt.Sprintf("`%d`", len(s.State.Guilds)),
					Inline: true,
				},
			}

		case "sistema":
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			embed.Fields = []*discordgo.MessageEmbedField{
				{
					Name:   "Uso de Memória",
					Value:  fmt.Sprintf("`%.2f MB`", float64(m.Alloc)/1024/1024),
					Inline: true,
				},
				{
					Name:   "Goroutines",
					Value:  fmt.Sprintf("`%d`", runtime.NumGoroutine()),
					Inline: true,
				},
				{
					Name:   "OS/Arquitetura",
					Value:  fmt.Sprintf("`%s/%s`", runtime.GOOS, runtime.GOARCH),
					Inline: true,
				},
				{
					Name:   "Versão Go",
					Value:  fmt.Sprintf("`%s`", runtime.Version()),
					Inline: true,
				},
				{
					Name:   "Núcleos da CPU",
					Value:  fmt.Sprintf("`%d`", runtime.NumCPU()),
					Inline: true,
				},
			}
		default:
			embed.Description = "Tipo inválido fornecido. Por favor, escolha 'basico', 'detalhado' ou 'sistema'."
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})

		return err
	},
}
