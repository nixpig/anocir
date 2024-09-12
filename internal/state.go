package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func GetState(containerID string) (*specs.State, error) {
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", containerID)

	stateJson, err := os.ReadFile(filepath.Join(containerPath, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("container not found")
		} else {
			return nil, fmt.Errorf("stat container path: %w", err)
		}
	}

	var state specs.State
	if err := json.Unmarshal(stateJson, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	return &state, nil
}

func SaveState(state *specs.State) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := os.WriteFile(
		filepath.Join(pkg.BrownieRootDir, "containers", state.ID, "state.json"),
		b,
		0644,
	); err != nil {
		return err
	}

	return nil
}
