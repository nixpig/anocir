// internal/cli/delete.go

package cli

import (
	"github.com/nixpig/anocir/internal/operations"
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

			return operations.Delete(&operations.DeleteOpts{
				ID:    containerID,
				Force: force,
			})
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Delete container regardless of state")

	return cmd
}
