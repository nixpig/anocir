package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
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

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			return cntr.DoWithLock(func(c *container.Container) error {
				state, err := c.GetState()
				if err != nil {
					return fmt.Errorf("failed to get container state: %w", err)
				}

				if _, err := fmt.Fprintln(cmd.OutOrStdout(), state); err != nil {
					return fmt.Errorf("failed to print state: %w", err)
				}

				return nil
			})
		},
	}

	return cmd
}
