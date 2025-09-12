package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [flags] CONTAINER_ID",
		Short: "Delete a container",
		Long:  "Release container resources after the container process has exited",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			if err := operations.Delete(&operations.DeleteOpts{
				ID:    containerID,
				Force: force,
			}); err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		BoolP("force", "f", false, "Delete container regardless of state")

	return cmd
}
