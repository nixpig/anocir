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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			force, _ := cmd.Flags().GetBool("force")
			rootDir, _ := cmd.Flags().GetString("root")

			if err := operations.Delete(&operations.DeleteOpts{
				ID:      containerID,
				Force:   force,
				RootDir: rootDir,
			}); err != nil {
				return fmt.Errorf("failed to delete container: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		BoolP("force", "f", false, "Force container deletion")

	return cmd
}
