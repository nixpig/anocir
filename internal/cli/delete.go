// internal/cli/delete.go

package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "delete [flags] CONTAINER_ID",
		Args: cobra.ExactArgs(1),
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
				logrus.Errorf("delete operation failed: %s", err)
				return fmt.Errorf("delete: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Delete container regardless of state")

	return cmd
}
