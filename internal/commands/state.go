package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/nixpig/brownie/internal/state"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StateOpts struct {
	ID string
}

type StateCLI struct {
	Version     string               `json:"ociVersion"`
	ID          string               `json:"id"`
	Status      specs.ContainerState `json:"status"`
	Pid         int                  `json:"pid,omitempty"`
	Bundle      string               `json:"bundle"`
	Rootfs      string               `json:"rootfs"`
	Annotations map[string]string    `json:"annotations,omitempty"`
}

func State(opts *StateOpts, log *zerolog.Logger) (string, error) {
	root := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)

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
