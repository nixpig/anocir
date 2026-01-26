package oci

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func pauseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pause [flags] CONTAINER_ID",
		Short:   "Pause a running container",
		Example: "  anocir pause busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, _ := cmd.Flags().GetString("root")

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			return cntr.DoWithLock(func(c *container.Container) error {
				if err := c.Pause(); err != nil {
					return fmt.Errorf("failed to pause container: %w", err)
				}

				return nil
			})
		},
	}

	return cmd
}
