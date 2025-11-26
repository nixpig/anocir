package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nixpig/anocir/internal/platform"
)

// Save persists the Container state to disk. It creates the required directory
// hierarchy and sets the needed permissions.
func (c *Container) Save() error {
	containerDir := filepath.Join(c.rootDir, c.State.ID)

	if c.spec.Linux != nil &&
		len(c.spec.Linux.UIDMappings) > 0 &&
		len(c.spec.Linux.GIDMappings) > 0 {
		if err := os.Chown(
			containerDir,
			int(c.spec.Linux.UIDMappings[0].HostID),
			int(c.spec.Linux.GIDMappings[0].HostID),
		); err != nil {
			return fmt.Errorf("chown container directory: %w", err)
		}
	}

	state, err := json.Marshal(c.State)
	if err != nil {
		return fmt.Errorf("serialise container state: %w", err)
	}

	stateFile := filepath.Join(containerDir, "state.json")

	if err := platform.AtomicWriteFile(stateFile, state, 0o644); err != nil {
		return fmt.Errorf("write container state: %w", err)
	}

	if c.pidFile != "" && c.State.Pid > 0 {
		if err := platform.AtomicWriteFile(
			c.pidFile,
			[]byte(strconv.Itoa(c.State.Pid)),
			0o644,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", c.pidFile, err)
		}
	}

	return nil
}
