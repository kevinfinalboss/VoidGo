package events

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/logger"
	"github.com/kevinfinalboss/Void/internal/types"
)

type Handler struct {
	events    map[string]*types.Event
	session   *discordgo.Session
	config    *config.Config
	logger    *logger.Logger
	readyOnce sync.Once
}

func NewHandler(s *discordgo.Session, cfg *config.Config, l *logger.Logger) *Handler {
	return &Handler{
		events:  make(map[string]*types.Event),
		session: s,
		config:  cfg,
		logger:  l,
	}
}

func (h *Handler) LoadEvents() error {
	h.logger.Info("Loading events...")
	err := h.loadEventFiles()
	if err != nil {
		return fmt.Errorf("error loading event files: %v", err)
	}

	h.registerDefaultHandlers()
	h.logger.Info("Events loaded successfully")
	return nil
}

func (h *Handler) loadEventFiles() error {
	return filepath.Walk("./events", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %v", path, err)
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") || strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %v", path, err)
		}

		if !strings.Contains(string(content), "var") || !strings.Contains(string(content), "Event") {
			return nil
		}

		packageName := getPackageName(string(content))
		if packageName == "" {
			return nil
		}

		h.logger.Info(fmt.Sprintf("Found event in %s", path))
		return nil
	})
}

func (h *Handler) registerDefaultHandlers() {
	h.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		h.readyOnce.Do(func() {
			h.logger.Info(fmt.Sprintf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator))
			h.logger.Info(fmt.Sprintf("Bot is in %d guilds", len(s.State.Guilds)))

			err := s.UpdateGameStatus(0, h.config.Discord.Status)
			if err != nil {
				h.logger.Error(fmt.Sprintf("Error setting status: %v", err))
			}
		})
	})

	h.session.AddHandler(func(s *discordgo.Session, e interface{}) {
		if evt, ok := e.(*discordgo.Event); ok {
			if evt.Type == "ERROR" {
				h.logger.Error("Discord error event occurred")
			}
		}
	})

	h.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		if h.config.Debug {
			h.logger.Info(fmt.Sprintf("Message received from %s: %s", m.Author.Username, m.Content))
		}
	})
}

func getPackageName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return ""
}
