package oci

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func execCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exec [flags] CONTAINER_ID COMMAND [args]",
		Short:   "execute a command in a container",
		Example: "  anocir exec busybox ps",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			execArgs := args[1:]

			rootDir, _ := cmd.Flags().GetString("root")
			cwd, _ := cmd.Flags().GetString("cwd")
			env, _ := cmd.Flags().GetStringSlice("env")
			additionalGIDs, _ := cmd.Flags().GetIntSlice("additional-gids")
			process, _ := cmd.Flags().GetString("process")
			processLabel, _ := cmd.Flags().GetString("process-label")
			appArmor, _ := cmd.Flags().GetString("apparmor")
			noNewPrivs, _ := cmd.Flags().GetBool("no-new-privs")
			capabilities, _ := cmd.Flags().GetStringSlice("cap")
			cgroup, _ := cmd.Flags().GetString("cgroup")
			consoleSocket, _ := cmd.Flags().GetString("console-socket")
			user, _ := cmd.Flags().GetString("user")
			pidFile, _ := cmd.Flags().GetString("pid-file")
			tty, _ := cmd.Flags().GetBool("tty")
			detach, _ := cmd.Flags().GetBool("detach")
			ignorePaused, _ := cmd.Flags().GetBool("ignore-paused")
			preserveFDs, _ := cmd.Flags().GetInt("preserve-fds")

			uid, gid, err := parseUser(user)
			if err != nil {
				return fmt.Errorf("parse user: %w", err)
			}

			if gid != 0 {
				additionalGIDs = append(additionalGIDs, gid)
			}

			envs, err := parseEnv(env)
			if err != nil {
				return fmt.Errorf("parse env: %w", err)
			}

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			return cntr.DoWithLock(func(c *container.Container) error {
				if err := container.Exec(
					&container.ExecOpts{
						ContainerPID:   c.State.Pid,
						Rootfs:         c.RootFS(),
						Cwd:            cwd,
						Args:           execArgs,
						Env:            envs,
						AdditionalGIDs: additionalGIDs,
						Process:        process,
						ProcessLabel:   processLabel,
						AppArmor:       appArmor,
						NoNewPrivs:     noNewPrivs,
						Capabilities:   capabilities,
						Cgroup:         cgroup,
						ConsoleSocket:  consoleSocket,
						UID:            uid,
						PIDFile:        pidFile,
						TTY:            tty,
						Detach:         detach,
						IgnorePaused:   ignorePaused,
						PreserveFDs:    preserveFDs,
					},
				); err != nil {
					return fmt.Errorf("failed to exec command: %w", err)
				}

				return nil
			})
		},
	}

	cmd.Flags().
		StringSliceP("env", "e", []string{}, "Set environment variable (name=value)")
	cmd.Flags().
		IntSliceP("additional-gids", "g", []int{}, "Additional GIDs")
	cmd.Flags().
		StringP("process", "p", "", "Path to process.json")
	cmd.Flags().
		String("process-label", "", "ASM process label")
	cmd.Flags().
		String("apparmor", "", "AppArmor profile for the process")
	cmd.Flags().
		Bool("no-new-privs", false, "Set no new privs")
	cmd.Flags().
		StringSlice("cap", []string{}, "Set capability")
	cmd.Flags().
		String("cgroup", "", "Specify cgroup (path | controller[,controller...]:path)")
	cmd.Flags().
		String("console-socket", "", "Console socket path")
	cmd.Flags().
		StringP("user", "u", "", "Run command as user uid[:gid]")
	cmd.Flags().
		String("pid-file", "", "File to write container PID to")
	cmd.Flags().
		BoolP("tty", "t", false, "Allocate a pseudo-terminal")
	cmd.Flags().
		BoolP("detach", "d", false, "Detach from container process")
	cmd.Flags().
		String("cwd", "", "Path in container to execute command")
	cmd.Flags().
		Bool("ignore-paused", false, "Allow exec in a paused container")
	cmd.Flags().
		Int("preserve-fds", 0, "Pass additional file descriptors to container")

	return cmd
}

func parseUser(u string) (int, int, error) {
	if u == "" {
		return 0, 0, nil
	}

	if strings.Contains(u, ":") {
		parts := strings.Split(u, ":")

		uid, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("parse UID (%s): %w", parts[0], err)
		}

		gid, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("parse GID (%s): %w", parts[1], err)
		}

		return uid, gid, nil
	}

	uid, err := strconv.Atoi(u)
	if err != nil {
		return 0, 0, fmt.Errorf("parse UID (%s): %w", u, err)
	}

	return uid, 0, nil
}

func parseEnv(env []string) (map[string]string, error) {
	envs := make(map[string]string)

	for _, e := range env {
		parts := strings.Split(e, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env %s", e)
		}

		envs[parts[0]] = parts[1]
	}

	return envs, nil
}
