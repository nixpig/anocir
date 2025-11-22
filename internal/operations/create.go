package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// CreateOpts holds the options for the Create operation.
type CreateOpts struct {
	// ID is the Container ID.
	ID string
	// Bundle is the location of the bundle.
	Bundle string
	// ConsoleSocket is the path of the unix domain socket where the Container
	// PTY master file descriptor is sent.
	ConsoleSocket string
	// PIDFile is an optional file location to write the PID of the container to.
	PIDFile string
	// RootDir is the directory to store the container state file.
	RootDir string
	// LogFile is the location of the file used for logging.
	LogFile string
}

// Create creates a new container.
func Create(opts *CreateOpts) error {
	// TODO: Validate Container ID is only alphanumeric, dash, underscore.
	if container.Exists(opts.ID, opts.RootDir) {
		return fmt.Errorf("container '%s' exists", opts.ID)
	}

	bundle, err := filepath.Abs(opts.Bundle)
	if err != nil {
		return fmt.Errorf("absolute path from bundle: %w", err)
	}

	config, err := os.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(config, &spec); err != nil {
		return fmt.Errorf("unmarshall config: %w", err)
	}

	// TODO: Validate spec.Version matches what the runtime supports.

	cntr, err := container.New(&container.ContainerOpts{
		ID:            opts.ID,
		Bundle:        bundle,
		Spec:          spec,
		ConsoleSocket: opts.ConsoleSocket,
		PIDFile:       opts.PIDFile,
		RootDir:       opts.RootDir,
		LogFile:       opts.LogFile,
	})
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := cntr.Save(); err != nil {
		return fmt.Errorf("save container: %w", err)
	}

	if err := cntr.Init(); err != nil {
		return fmt.Errorf("initialise container: %w", err)
	}

	return nil
}
