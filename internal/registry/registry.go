package registry

import (
	"github.com/kevinfinalboss/Void/internal/types"
)

var Commands = make(map[string]*types.Command)

func RegisterCommand(cmd *types.Command) {
	Commands[cmd.Name] = cmd
}
