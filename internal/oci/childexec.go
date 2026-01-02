package oci

import (
	"fmt"

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
			execArgs, _ := cmd.Flags().GetStringSlice("args")
			_, _ = cmd.Flags().GetStringSlice("envs")
			caps, _ := cmd.Flags().GetStringSlice("caps")
			additionalGIDs, _ := cmd.Flags().GetIntSlice("additional-gids")
			noNewPrivs, _ := cmd.Flags().GetBool("no-new-privs")

			user := &specs.User{
				UID: uint32(uid),
				GID: uint32(gid),
			}

			for _, g := range additionalGIDs {
				user.AdditionalGids = append(user.AdditionalGids, uint32(g))
			}

			if err := container.ChildExec(&container.ChildExecOpts{
				Cwd:          cwd,
				Args:         execArgs,
				User:         user,
				Capabilities: &specs.LinuxCapabilities{Bounding: caps},
				NoNewPrivs:   noNewPrivs,
			}); err != nil {
				return fmt.Errorf("fork/exec child: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().String("cwd", "", "")
	cmd.Flags().Int("uid", 0, "")
	cmd.Flags().Int("gid", 0, "")
	cmd.Flags().StringSlice("args", []string{}, "")
	cmd.Flags().StringSlice("envs", []string{}, "")
	cmd.Flags().StringSlice("caps", []string{}, "")
	cmd.Flags().IntSlice("additional-gids", []int{}, "")
	cmd.Flags().Bool("no-new-privs", false, "")

	return cmd
}
