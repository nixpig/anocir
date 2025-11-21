package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func stateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Get the state of a container",
		Example: "  anocir state busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, _ := cmd.Flags().GetString("root")

			state, err := operations.State(&operations.StateOpts{
				ID:      containerID,
				RootDir: rootDir,
			})
			if err != nil {
				return fmt.Errorf("failed to get state of container: %w", err)
			}

			if _, err := cmd.OutOrStdout().Write([]byte(state)); err != nil {
				return fmt.Errorf("failed to print state to stdout: %w", err)
			}

			return nil
		},
	}

	return cmd
}
