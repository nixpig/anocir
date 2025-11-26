package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// Load retrieves an existing Container with the given id at the given rootDir.
func Load(id, rootDir string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(rootDir, id, "state.json"))
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state *specs.State
	if err := json.Unmarshal(s, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	config, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(config, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	c := &Container{
		State:         state,
		spec:          spec,
		RootDir:       rootDir,
		containerSock: filepath.Join(rootDir, id, containerSockFilename),
	}

	return c, nil
}

// Exists checks if a container exists with the given id at the given rootDir.
func Exists(id, rootDir string) bool {
	_, err := os.Stat(filepath.Join(rootDir, id))

	return err == nil
}
