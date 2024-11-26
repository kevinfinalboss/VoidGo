package ready

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/internal/types"
)

var ReadyEvent = &types.Event{
	Name: "ready",
	Handler: func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("Bot is ready! Logged in as %s#%s\n", s.State.User.Username, s.State.User.Discriminator)
	},
}
