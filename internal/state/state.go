package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const stateFilename = "state.json"

type State struct {
	Version     string
	ID          string
	Bundle      string
	Annotations map[string]string
	Status      specs.ContainerState
	PID         int
}

func New(
	id string,
	bundle string,
	status specs.ContainerState,
) *State {
	return &State{
		Version:     pkg.OCIVersion,
		ID:          id,
		Bundle:      bundle,
		Annotations: map[string]string{},
		Status:      status,
	}
}

func Load(root string) (*State, error) {
	b, err := os.ReadFile(
		filepath.Join(root, stateFilename),
	)
	if err != nil {
		return nil, fmt.Errorf("read container state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}

	return &state, nil
}

func (s *State) Save(root string) error {
	state, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if err := os.WriteFile(
		filepath.Join(root, stateFilename),
		state,
		0644,
	); err != nil {
		return err
	}

	return nil
}
