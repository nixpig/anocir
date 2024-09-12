package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func GetState(containerID string) (*specs.State, error) {
	containerPath := filepath.Join(BrownieRootDir, "containers", containerID)

	fc, err := os.ReadFile(filepath.Join(containerPath, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("container not found")
		} else {
			return nil, fmt.Errorf("stat container path: %w", err)
		}
	}

	var state specs.State
	if err := json.Unmarshal(fc, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	return &state, nil
}
