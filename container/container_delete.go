package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func (c *Container) Delete(force bool) error {
	if !force && !c.CanBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state (%s)", c.Status())
	}

	process, err := os.FindProcess(c.PID())
	if err != nil {
		return fmt.Errorf("find container process (%d): %w", c.PID(), err)
	}
	if process != nil {
		process.Signal(syscall.Signal(9))
	}

	if err := os.RemoveAll(filepath.Join(containerRootDir, c.ID())); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	if err := c.ExecHooks("poststop"); err != nil {
		fmt.Println("Warning: failed to execute poststop hooks")
	}

	return nil
}

func killAllChildren(pid int) error {
	childPIDs, err := findChildPIDs(pid)
	if err != nil {
		return fmt.Errorf("find child pids: %w", err)
	}

	for _, p := range childPIDs {
		if err := syscall.Kill(p, syscall.Signal(9)); err != nil {
			return fmt.Errorf("kill child pid: %w", err)
		}
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
