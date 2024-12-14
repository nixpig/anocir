package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nixpig/brownie/container"
)

type StateOpts struct {
	ID string
}

func State(opts *StateOpts) (string, error) {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return "", err
	}

	if err := cntr.RefreshState(); err != nil {
		return "", err
	}

	s, err := json.Marshal(cntr.State)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(s), nil
}
