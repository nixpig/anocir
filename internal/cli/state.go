package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func stateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Get container state",
		Long:    "Request the state of the container",
		Example: "  anocir state busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, err := cmd.Flags().GetString("root")
			if err != nil {
				return err
			}

			state, err := operations.State(&operations.StateOpts{
				ID:      containerID,
				RootDir: rootDir,
			})
			if err != nil {
				return err
			}

			if _, err := cmd.OutOrStdout().Write([]byte(state)); err != nil {
				return fmt.Errorf("write state to stdout: %w", err)
			}

			return nil
		},
	}

	return cmd
}
