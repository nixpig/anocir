package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"

	"github.com/nixpig/anocir/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
)

func childExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "childexec [flags]",
		Short:  internalUseMessage,
		Args:   cobra.NoArgs,
		Hidden: true, // this command is only used internally
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := cmd.Flags().GetString("cwd")
			uid, _ := cmd.Flags().GetInt("uid")
			gid, _ := cmd.Flags().GetInt("gid")
			execArgs, _ := cmd.Flags().GetStringArray("args")
			envs, _ := cmd.Flags().GetStringArray("envs")
			caps, _ := cmd.Flags().GetStringArray("caps")
			additionalGIDs, _ := cmd.Flags().GetIntSlice("additional-gids")
			noNewPrivs, _ := cmd.Flags().GetBool("no-new-privs")
			tty, _ := cmd.Flags().GetBool("tty")
			containerID, _ := cmd.Flags().GetString("container-id")
			appArmorProfile, _ := cmd.Flags().GetString("apparmor")
			processLabel, _ := cmd.Flags().GetString("process-label")

			user := &specs.User{UID: uint32(uid), GID: uint32(gid)}

			for _, g := range additionalGIDs {
				user.AdditionalGids = append(user.AdditionalGids, uint32(g))
			}

			var seccomp *specs.LinuxSeccomp
			seccompFD := os.Getenv(container.EnvSeccompFD)
			if seccompFD != "" {
				seccompFDNum, err := strconv.Atoi(seccompFD)
				if err != nil {
					return fmt.Errorf("convert seccomp fd number: %w", err)
				}

				seccompFile := os.NewFile(uintptr(seccompFDNum), "seccomp")
				data, err := io.ReadAll(seccompFile)
				if err != nil {
					return fmt.Errorf("read seccomp file: %w", err)
				}
				if err := seccompFile.Close(); err != nil {
					slog.Warn("failed to close seccomp file", "container_id", containerID, "err", err)
				}

				seccomp = &specs.LinuxSeccomp{}
				if err := json.Unmarshal(data, seccomp); err != nil {
					return fmt.Errorf("parse seccomp profile: %w", err)
				}
			}

			if err := container.ChildExec(&container.ChildExecOpts{
				Cwd:             cwd,
				Args:            execArgs,
				Env:             envs,
				User:            user,
				Capabilities:    &specs.LinuxCapabilities{Bounding: caps},
				NoNewPrivs:      noNewPrivs,
				TTY:             tty,
				ContainerID:     containerID,
				Seccomp:         seccomp,
				AppArmorProfile: appArmorProfile,
				ProcessLabel:    processLabel,
			}); err != nil {
				return fmt.Errorf("fork/exec child: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().String("cwd", "", "")
	cmd.Flags().Int("uid", 0, "")
	cmd.Flags().Int("gid", 0, "")
	cmd.Flags().StringArray("args", []string{}, "")
	cmd.Flags().StringArray("envs", []string{}, "")
	cmd.Flags().StringArray("caps", []string{}, "")
	cmd.Flags().IntSlice("additional-gids", []int{}, "")
	cmd.Flags().Bool("no-new-privs", false, "")
	cmd.Flags().Bool("tty", false, "")
	cmd.Flags().String("container-id", "", "")
	cmd.Flags().String("apparmor", "", "")
	cmd.Flags().String("process-label", "", "")

	return cmd
}
