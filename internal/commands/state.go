package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/internal/state"
	"github.com/rs/zerolog"
)

type StateOpts struct {
	ID string
}

func State(opts *StateOpts, log *zerolog.Logger) (string, error) {
	root := container.GetRoot(opts.ID)

	state, err := state.Load(root)
	if err != nil {
		return "", fmt.Errorf("load container: %w", err)
	}

	s, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(s), nil
}
