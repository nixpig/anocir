package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const BrownieRootDir = "/var/lib/brownie"

type status string

const (
	Creating status = "creating"
	Created  status = "created"
	Running  status = "running"
	Stopped  status = "stopped"
)

type State struct {
	OCIVersion  string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      status            `json:"status"`
	PID         *int              `json:"pid,omitempty"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

func GetState(containerID string) (*State, error) {
	containerPath := filepath.Join(BrownieRootDir, "containers", containerID)

	fc, err := os.ReadFile(filepath.Join(containerPath, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("container not found")
		} else {
			return nil, fmt.Errorf("stat container path: %w", err)
		}
	}

	var state State
	if err := json.Unmarshal(fc, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	return &state, nil
}
