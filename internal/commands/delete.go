package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger) error {
	state, err := internal.GetState(opts.ID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	if !opts.Force && state.Status != specs.StateStopped {
		return errors.New("container is not stopped")
	}

	if err := os.Remove(filepath.Join(pkg.BrownieRootDir, "containers", state.ID, "container.sock")); err != nil {
		return fmt.Errorf("remove ipc socket: %w", err)
	}

	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)
	if err := os.RemoveAll(containerPath); err != nil {
		return fmt.Errorf("remove container path: %s", err)
	}

	configJSON, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	// 13. Invoke poststop hooks
	// FIXME: ?? config should probably be initially copied across, since any subsequent changes to poststop hooks will get picked up here when they shouldn't
	// See: Any updates to config.json after this step MUST NOT affect the container.
	if spec.Hooks != nil {
		if err := internal.ExecHooks(spec.Hooks.Poststop); err != nil {
			return fmt.Errorf("execute poststop hooks: %w", err)
		}
	}

	return nil
}
