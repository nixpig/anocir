package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func killCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kill [flags] CONTAINER_ID SIGNAL",
		Short:   "Send signal to a container",
		Long:    "Send a signal to the container process",
		Example: "  anocir kill busybox 9",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			signal := args[1]

			rootDir, _ := cmd.Flags().GetString("root")

			if err := operations.Kill(&operations.KillOpts{
				ID:      containerID,
				Signal:  signal,
				RootDir: rootDir,
			}); err != nil {
				return fmt.Errorf("kill: %w", err)
			}

			return nil
		},
	}

	// TODO: figure out why Docker needs this and implement it
	cmd.Flags().BoolP("all", "a", false, "Kill all (Docker??)")

	return cmd
}
