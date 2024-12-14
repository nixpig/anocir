package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Kill(sig syscall.Signal) error {
	if !c.CanBeKilled() {
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
		return fmt.Errorf("failed to execute kill syscall (process: %d): %w", c.PID(), err)
	}

	c.SetStatus(specs.StateStopped)
	if err := c.Save(); err != nil {
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	// TODO: delete everything then
	if err := c.ExecHooks("poststop"); err != nil {
		// TODO: log a warning???
		fmt.Println("Warning: failed to execute poststop hooks")
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
