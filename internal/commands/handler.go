package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/kevinfinalboss/Void/commands/all"
	"github.com/kevinfinalboss/Void/config"
	"github.com/kevinfinalboss/Void/internal/logger"
	"github.com/kevinfinalboss/Void/internal/registry"
	"github.com/kevinfinalboss/Void/internal/types"
)

type Handler struct {
	commands     map[string]*types.Command
	session      *discordgo.Session
	config       *config.Config
	logger       *logger.Logger
	cooldowns    sync.Map
	commandMutex sync.RWMutex
}

func NewHandler(s *discordgo.Session, cfg *config.Config, l *logger.Logger) *Handler {
	return &Handler{
		commands: make(map[string]*types.Command),
		session:  s,
		config:   cfg,
		logger:   l,
	}
}

func (h *Handler) LoadCommands() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	startTime := time.Now()
	h.logger.Info("Loading commands...")

	cmdList := make([]*types.Command, 0, len(registry.Commands))
	nameMap := make(map[*types.Command]string)
	for name, cmd := range registry.Commands {
		cmdList = append(cmdList, cmd)
		nameMap[cmd] = name
	}

	deleteCtx, deleteCancel := context.WithTimeout(ctx, 30*time.Second)
	defer deleteCancel()

	deleteDone := make(chan error, 1)
	go func() {
		deleteDone <- h.DeleteCommands()
		close(deleteDone)
	}()

	select {
	case err := <-deleteDone:
		if err != nil {
			h.logger.Error(fmt.Sprintf("Error deleting commands: %v", err))
		}
	case <-deleteCtx.Done():
		return fmt.Errorf("command deletion timed out")
	}

	numCommands := len(cmdList)
	if numCommands == 0 {
		return nil
	}

	workerCount := min(5, numCommands)
	batchSize := (numCommands + workerCount - 1) / workerCount

	results := make(chan error, numCommands)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		start := i * batchSize
		if start >= numCommands {
			break
		}

		end := min(start+batchSize, numCommands)
		cmds := cmdList[start:end]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, cmd := range cmds {
				select {
				case <-ctx.Done():
					results <- ctx.Err()
					return
				default:
					if err := h.registerCommand(nameMap[cmd], cmd); err != nil {
						results <- err
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var errors []error
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to register commands: %v", errors)
	}

	h.logger.Info(fmt.Sprintf("Successfully loaded %d commands in %v", numCommands, time.Since(startTime)))
	return nil
}

func (h *Handler) registerCommand(name string, cmd *types.Command) error {
	options := make([]*discordgo.ApplicationCommandOption, 0, len(cmd.Options))
	for _, opt := range cmd.Options {
		options = append(options, &discordgo.ApplicationCommandOption{
			Name:        opt.Name,
			Description: opt.Description,
			Type:        opt.Type,
			Required:    opt.Required,
			Choices:     opt.Choices,
		})
	}

	command := &discordgo.ApplicationCommand{
		Name:        cmd.Name,
		Description: cmd.Description,
		Options:     options,
	}

	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		_, err = h.session.ApplicationCommandCreate(
			h.config.Discord.ClientID,
			h.config.Discord.GuildID,
			command,
		)
		if err == nil {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to register command %s: %v", name, err)
	}

	h.commandMutex.Lock()
	h.commands[name] = cmd
	h.commandMutex.Unlock()

	return nil
}

func (h *Handler) DeleteCommands() error {
	commands, err := h.session.ApplicationCommands(h.config.Discord.ClientID, h.config.Discord.GuildID)
	if err != nil {
		return fmt.Errorf("error fetching commands: %v", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(commands))

	for _, cmd := range commands {
		wg.Add(1)
		go func(cmd *discordgo.ApplicationCommand) {
			defer wg.Done()
			for i := 0; i < 3; i++ {
				err := h.session.ApplicationCommandDelete(
					h.config.Discord.ClientID,
					h.config.Discord.GuildID,
					cmd.ID,
				)
				if err == nil {
					break
				}
				if i == 2 {
					errChan <- fmt.Errorf("failed to delete command %s: %v", cmd.Name, err)
				}
				time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
			}
		}(cmd)
	}

	wg.Wait()
	close(errChan)

	var errorList []error
	for err := range errChan {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		return fmt.Errorf("errors deleting commands: %v", errorList)
	}

	return nil
}

func (h *Handler) HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		h.handleAutocomplete(s, i)
		return
	}

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	userID := h.getUserID(i)
	if userID == "" {
		h.logger.Error("Não foi possível identificar o usuário na interação.")
		return
	}

	commandName := i.ApplicationCommandData().Name
	h.commandMutex.RLock()
	cmd, exists := h.commands[commandName]
	h.commandMutex.RUnlock()

	if !exists {
		h.logger.Error(fmt.Sprintf("Command not found: %s", commandName))
		return
	}

	if !h.checkCooldown(userID, commandName) {
		h.respondWithCooldownMessage(s, i)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run(s, i, h.config)
	}()

	select {
	case err := <-done:
		if err != nil {
			h.handleError(s, i, commandName, err)
		}
	case <-ctx.Done():
		h.handleError(s, i, commandName, fmt.Errorf("command execution timed out"))
	}
}

func (h *Handler) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	commandName := i.ApplicationCommandData().Name
	h.commandMutex.RLock()
	cmd, exists := h.commands[commandName]
	h.commandMutex.RUnlock()

	if !exists || cmd.AutoComplete == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		choices, err := cmd.AutoComplete(s, i)
		if err != nil {
			h.logger.Error(fmt.Sprintf("Error in autocomplete for command %s: %v", commandName, err))
			return
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: choices,
			},
		})
		if err != nil {
			h.logger.Error(fmt.Sprintf("Error sending autocomplete response: %v", err))
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
		h.logger.Error("Autocomplete timed out")
	}
}

func (h *Handler) getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (h *Handler) checkCooldown(userID, commandName string) bool {
	key := fmt.Sprintf("%s:%s", commandName, userID)
	now := time.Now().Unix()

	if lastUsage, exists := h.cooldowns.Load(key); exists {
		if now-lastUsage.(int64) < 5 {
			return false
		}
	}

	h.cooldowns.Store(key, now)
	return true
}

func (h *Handler) respondWithCooldownMessage(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Por favor, aguarde antes de usar este comando novamente.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) handleError(s *discordgo.Session, i *discordgo.InteractionCreate, commandName string, err error) {
	h.logger.Error(fmt.Sprintf("Error executing command %s: %v", commandName, err))
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Ocorreu um erro ao executar o comando.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
