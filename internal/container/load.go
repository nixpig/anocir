package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// Exists checks if a container exists with the given id at the given rootDir.
func Exists(id, rootDir string) bool {
	_, err := os.Stat(filepath.Join(rootDir, id))

	return err == nil
}

// Load retrieves an existing container with the given id at the given rootDir.
func Load(id, rootDir string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(rootDir, id, "state.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("container %s does not exist", id)
		}

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
		containerSock: containerSockPath(state.Bundle),
	}

	return c, nil
}

// containerSockPath constructs the filepath to the socket used for container IPC.
func containerSockPath(bundle string) string {
	// We use a path that's always accessible by the runtime, rather than the
	// root or bundle paths.
	// TODO: Review how we handle this.
	return filepath.Join("/run/anocir", ipc.ShortID(bundle), containerSockFilename)
}
