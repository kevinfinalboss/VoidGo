package guild

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/internal/database"
	"github.com/kevinfinalboss/Void/internal/logger"
	"github.com/kevinfinalboss/Void/internal/models"
)

type Handler struct {
	db     *database.MongoDB
	logger *logger.Logger
}

func NewHandler(db *database.MongoDB, logger *logger.Logger) *Handler {
	return &Handler{
		db:     db,
		logger: logger,
	}
}

func (h *Handler) HandleGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	guild := &models.Guild{
		GuildID:     g.ID,
		Name:        g.Name,
		OwnerID:     g.OwnerID,
		MemberCount: g.MemberCount,
		IsActive:    true,
		JoinedAt:    time.Now(),
		Region:      g.Region,
		Icon:        g.Icon,
		Features:    g.Features,
		LastUpdated: time.Now(),
	}

	if err := h.db.UpsertGuild(guild); err != nil {
		h.logger.Error("Failed to upsert guild:", err)
		return
	}
}

func (h *Handler) HandleGuildDelete(s *discordgo.Session, g *discordgo.GuildDelete) {
	now := time.Now()
	if err := h.db.UpdateGuildStatus(g.ID, false, &now); err != nil {
		h.logger.Error("Failed to update guild status:", err)
		return
	}

	h.logger.Info("Bot removed from guild ID:", g.ID)
}

func (h *Handler) HandleGuildUpdate(s *discordgo.Session, g *discordgo.GuildUpdate) {
	guild := &models.Guild{
		GuildID:     g.ID,
		Name:        g.Name,
		OwnerID:     g.OwnerID,
		MemberCount: g.MemberCount,
		IsActive:    true,
		Region:      g.Region,
		Icon:        g.Icon,
		Features:    g.Features,
		LastUpdated: time.Now(),
	}

	if err := h.db.UpsertGuild(guild); err != nil {
		h.logger.Error("Failed to update guild:", err)
		return
	}

	h.logger.Info("Guild updated:", guild.Name, "(ID:", guild.GuildID, ")")
}

func (h *Handler) HandleGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if err := h.db.UpdateMemberCount(m.GuildID, 1); err != nil {
		h.logger.Error("Failed to update member count:", err)
	}
}

func (h *Handler) HandleGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	if err := h.db.UpdateMemberCount(m.GuildID, -1); err != nil {
		h.logger.Error("Failed to update member count:", err)
	}
}
