package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal/bundle"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type ContainerState struct {
	Path string
	specs.State
}

func NewState(id string, bundle *bundle.Bundle) (*ContainerState, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id, "state.json")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create state file: %w", err)
	}
	if f != nil {
		f.Close()
	}

	return &ContainerState{
		Path: path,
		State: specs.State{
			Version:     bundle.Spec.Version,
			ID:          id,
			Bundle:      bundle.Path,
			Annotations: bundle.Spec.Annotations,
		},
	}, nil
}

func LoadState(id string) (*ContainerState, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id, "state.json")

	stateJSON, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read container state file: %w", err)
	}

	var state ContainerState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}

	return &state, nil
}

func (c *ContainerState) Set(status specs.ContainerState) {
	c.Status = status
}

func (c *ContainerState) Save() error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.Path, b, 0644); err != nil {
		return err
	}

	return nil
}
