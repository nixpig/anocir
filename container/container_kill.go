package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Kill(sig syscall.Signal, log *zerolog.Logger) error {
	if !c.CanBeKilled() {
		log.Error().Str("state", string(c.State.Status)).Msg("container cannot be killed in current state")
		return fmt.Errorf("container cannot be killed in current state: %s", c.Status())
	}

	childPIDs, err := findChildPIDs(c.PID())
	if err != nil {
		return fmt.Errorf("find child pids: %w", err)
	}

	// FIXME: shouldn't need to recursively kill child pids because the process should 'replace' the parent pid, a'la execve
	for _, p := range childPIDs {
		if err := syscall.Kill(p, sig); err != nil {
			return fmt.Errorf("kill child pid: %w", err)
		}
	}

	if err := syscall.Kill(c.PID(), sig); err != nil {
		log.Error().Err(err).Int("pid", c.PID()).Msg("failed to execute kill syscall")
		return fmt.Errorf("failed to execute kill syscall (process: %d): %w", c.PID(), err)
	}

	c.SetStatus(specs.StateStopped)
	if err := c.HSave(); err != nil {
		log.Error().Err(err).Msg("failed to save stopped state")
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	// TODO: delete everything then
	if err := c.ExecHooks("poststop"); err != nil {
		log.Error().Err(err).Msg("failed to execute poststop hooks")
		fmt.Println("failed to execute poststop hooks")
		// TODO: log a warning???
	}

	return nil
}

func findChildPIDs(parentPID int) ([]int, error) {
	var childPIDs []int

	var findDescendants func(int)
	findDescendants = func(pid int) {
		procDirs, err := os.ReadDir("/proc")
		if err != nil {
			return
		}

		for _, procDir := range procDirs {
			if !procDir.IsDir() {
				continue
			}

			childPid, err := strconv.Atoi(procDir.Name())
			if err != nil {
				continue
			}

			statusPath := filepath.Join("/proc", procDir.Name(), "status")
			statusBytes, err := os.ReadFile(statusPath)
			if err != nil {
				continue
			}

			status := string(statusBytes)
			for _, line := range strings.Split(status, "\n") {
				if strings.HasPrefix(line, "PPid:") {
					fields := strings.Fields(line)
					if len(fields) == 2 {
						ppid, err := strconv.Atoi(fields[1])
						if err != nil {
							break
						}
						if ppid == pid {
							childPIDs = append(childPIDs, childPid)
							findDescendants(childPid)
						}
					}
					break
				}
			}
		}
	}

	findDescendants(parentPID)

	return childPIDs, nil
}
