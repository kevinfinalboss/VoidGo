package registry

import (
	"github.com/kevinfinalboss/Void/internal/types"
)

// Mapa global para armazenar comandos registrados
var Commands = make(map[string]*types.Command)

// RegisterCommand registra um comando no mapa global
func RegisterCommand(cmd *types.Command) {
	Commands[cmd.Name] = cmd
}
