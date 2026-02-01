package oci

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
)

func childExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "childexec [flags]",
		Short:  "\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
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
			seccompFile, _ := cmd.Flags().GetString("seccomp-file")

			user := &specs.User{
				UID: uint32(uid),
				GID: uint32(gid),
			}

			for _, g := range additionalGIDs {
				user.AdditionalGids = append(user.AdditionalGids, uint32(g))
			}

			// TODO: Not really happy about passing the seccomp profile via a file
			// that we delete after reading. Find a cleaner way.
			var seccomp *specs.LinuxSeccomp
			if seccompFile != "" {
				data, err := os.ReadFile(seccompFile)
				if err != nil {
					return fmt.Errorf("read seccomp file: %w", err)
				}
				os.Remove(seccompFile)

				seccomp = &specs.LinuxSeccomp{}
				if err := json.Unmarshal(data, seccomp); err != nil {
					return fmt.Errorf("parse seccomp profile: %w", err)
				}
			}

			if err := container.ChildExec(&container.ChildExecOpts{
				Cwd:          cwd,
				Args:         execArgs,
				Env:          envs,
				User:         user,
				Capabilities: &specs.LinuxCapabilities{Bounding: caps},
				NoNewPrivs:   noNewPrivs,
				TTY:          tty,
				ContainerID:  containerID,
				Seccomp:      seccomp,
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
	cmd.Flags().String("seccomp-file", "", "")

	return cmd
}
