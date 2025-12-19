package oci

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [flags] CONTAINER_ID",
		Short: "Delete a container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			force, _ := cmd.Flags().GetBool("force")
			rootDir, _ := cmd.Flags().GetString("root")

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			return cntr.DoWithLock(func(c *container.Container) error {
				if err := c.Delete(force); err != nil {
					return fmt.Errorf("failed to delete container: %w", err)
				}
				return nil
			})
		},
	}

	cmd.Flags().
		BoolP("force", "f", false, "Force container deletion")

	return cmd
}
