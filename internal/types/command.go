package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
)

type CommandOption struct {
	Name        string
	Description string
	Type        discordgo.ApplicationCommandOptionType
	Required    bool
	Choices     []*discordgo.ApplicationCommandOptionChoice
}

type Command struct {
	Name         string
	Description  string
	Category     string
	Cooldown     time.Duration
	AllowPrefix  bool
	DevOnly      bool
	AdminOnly    bool
	CommandType  discordgo.ApplicationCommandType
	Options      []*CommandOption
	AutoComplete func(s *discordgo.Session, i *discordgo.InteractionCreate) ([]*discordgo.ApplicationCommandOptionChoice, error)
	Run          func(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) error
}
