package oci

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nixpig/anocir/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func execCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exec [flags] CONTAINER_ID COMMAND [args]",
		Short:   "Execute a command in a container",
		Example: "  anocir exec busybox ps",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, _ := cmd.Flags().GetString("root")
			process, _ := cmd.Flags().GetString("process")
			consoleSocket, _ := cmd.Flags().GetString("console-socket")
			pidFile, _ := cmd.Flags().GetString("pid-file")
			detach, _ := cmd.Flags().GetBool("detach")
			preserveFDs, _ := cmd.Flags().GetInt("preserve-fds")
			cgroup, _ := cmd.Flags().GetString("cgroup")
			ignorePaused, _ := cmd.Flags().GetBool("ignore-paused")

			opts := &container.ExecOpts{
				Cgroup:        cgroup,
				ConsoleSocket: consoleSocket,
				PIDFile:       pidFile,
				Detach:        detach,
				IgnorePaused:  ignorePaused,
				PreserveFDs:   preserveFDs,
				ContainerID:   containerID,
			}

			if process != "" {
				if err := parseProcessFile(opts, process); err != nil {
					return fmt.Errorf("failed to parse process file: %w", err)
				}
			} else {
				if err := parseProcessFlags(opts, cmd.Flags(), args); err != nil {
					return fmt.Errorf("failed to parse process flags: %w", err)
				}
			}

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			var execErr error
			var execExit int

			if err := cntr.DoWithLock(func(c *container.Container) error {
				execExit, execErr = container.Exec(c.State.Pid, opts)
				return nil
			}); err != nil {
				return fmt.Errorf("failed to lock container: %w", err)
			}

			// os.Exit calls need to happen outside of DoWithLock callback to ensure
			// the defered Unlock gets called before exiting.
			if execErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", execErr)
				if execExit != 0 {
					os.Exit(execExit)
				}
				os.Exit(255)
			}

			return nil
		},
	}

	cmd.Flags().
		StringArrayP("env", "e", []string{}, "set environment variable (name=value)")
	cmd.Flags().
		IntSliceP("additional-gids", "g", []int{}, "additional GIDs")
	cmd.Flags().
		StringP("process", "p", "", "path to process.json")
	cmd.Flags().
		String("process-label", "", "ASM process label")
	cmd.Flags().
		String("apparmor", "", "AppArmor profile for the process")
	cmd.Flags().
		Bool("no-new-privs", false, "set no new privs")
	cmd.Flags().
		StringArray("cap", []string{}, "set capabilities")
	cmd.Flags().
		String("cgroup", "", "cgroup (path | controller[,controller...]:path)")
	cmd.Flags().
		String("console-socket", "", "console socket path")
	cmd.Flags().
		StringP("user", "u", "", "run command as user uid[:gid]")
	cmd.Flags().
		String("pid-file", "", "file to write container PID to")
	cmd.Flags().
		BoolP("tty", "t", false, "allocate a pseudo-terminal")
	cmd.Flags().
		BoolP("detach", "d", false, "detach from container process")
	cmd.Flags().
		String("cwd", "", "path in container to execute command")
	cmd.Flags().
		Bool("ignore-paused", false, "allow exec in a paused container")
	cmd.Flags().
		Int("preserve-fds", 0, "pass additional file descriptors to container")

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

func parseProcessFile(opts *container.ExecOpts, process string) error {
	data, err := os.ReadFile(process)
	if err != nil {
		return fmt.Errorf("read process file: %w", err)
	}

	var p specs.Process
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse process JSON: %w", err)
	}

	opts.Cwd = p.Cwd
	opts.Env = p.Env
	opts.Args = p.Args
	opts.UID = int(p.User.UID)
	opts.GID = int(p.User.GID)
	opts.NoNewPrivs = p.NoNewPrivileges
	opts.AppArmor = p.ApparmorProfile
	opts.TTY = p.Terminal
	opts.ProcessLabel = p.SelinuxLabel

	if p.Capabilities != nil {
		opts.Capabilities = p.Capabilities.Bounding
	}

	opts.AdditionalGIDs = make([]int, 0, len(p.User.AdditionalGids))
	for _, g := range p.User.AdditionalGids {
		opts.AdditionalGIDs = append(opts.AdditionalGIDs, int(g))
	}

	return nil
}

func parseProcessFlags(
	opts *container.ExecOpts,
	flags *pflag.FlagSet,
	args []string,
) error {
	if len(args) > 1 {
		opts.Args = args[1:]
	}

	opts.Cwd, _ = flags.GetString("cwd")
	opts.Env, _ = flags.GetStringArray("env")
	opts.Capabilities, _ = flags.GetStringArray("cap")
	opts.NoNewPrivs, _ = flags.GetBool("no-new-privs")
	opts.AppArmor, _ = flags.GetString("apparmor")
	opts.TTY, _ = flags.GetBool("tty")
	opts.ProcessLabel, _ = flags.GetString("process-label")

	user, _ := flags.GetString("user")

	var err error
	opts.UID, opts.GID, err = parseUser(user)
	if err != nil {
		return fmt.Errorf("parse user: %w", err)
	}

	additionalGIDs, _ := flags.GetIntSlice("additional-gids")
	opts.AdditionalGIDs = append(opts.AdditionalGIDs, additionalGIDs...)

	return nil
}
