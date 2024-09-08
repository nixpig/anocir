package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/nixpig/brownie/pkg/config"
)

func Delete(containerID string) error {
	containerPath := filepath.Join(BrownieRootDir, "containers", containerID)

	fc, err := os.ReadFile(filepath.Join(containerPath, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("container not found")
		} else {
			return fmt.Errorf("stat container path: %w", err)
		}
	}

	var state pkg.State
	if err := json.Unmarshal(fc, &state); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	if state.Status != pkg.Stopped {
		return errors.New("container is not stopped")
	}

	if err := os.Remove(fmt.Sprintf("/tmp/brownie_%s.sock", state.ID)); err != nil {
		return fmt.Errorf("remove ipc socket: %w", err)
	}

	if err := os.RemoveAll(containerPath); err != nil {
		return fmt.Errorf("remove container path: %s", err)
	}

	c, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(c, &cfg); err != nil {
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	// 13. Invoke poststop hooks
	// FIXME: ?? config should probably be initially copied across, since any subsequent changes to poststop hooks will get picked up here when they shouldn't
	// See: Any updates to config.json after this step MUST NOT affect the container.
	if err := internal.ExecHooks(cfg.Hooks.PostStop); err != nil {
		return fmt.Errorf("execute poststop hooks: %w", err)
	}

	return nil
}
