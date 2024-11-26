package models

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Guild struct {
	ID          primitive.ObjectID       `bson:"_id,omitempty"`
	GuildID     string                   `bson:"guild_id"`
	Name        string                   `bson:"name"`
	OwnerID     string                   `bson:"owner_id"`
	MemberCount int                      `bson:"member_count"`
	IsActive    bool                     `bson:"is_active"`
	JoinedAt    time.Time                `bson:"joined_at"`
	LeftAt      *time.Time               `bson:"left_at,omitempty"`
	Region      string                   `bson:"region"`
	Icon        string                   `bson:"icon"`
	Features    []discordgo.GuildFeature `bson:"features"`
	LastUpdated time.Time                `bson:"last_updated"`
	Settings    GuildSettings            `bson:"settings"`
}

type GuildSettings struct {
	AuditLogChannel string `bson:"audit_log_channel"`
}
