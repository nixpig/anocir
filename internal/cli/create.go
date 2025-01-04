// internal/cli/create.go

package cli

import (
	"os"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create [flags] CONTAINER_ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			bundle, err := cmd.Flags().GetString("bundle")
			if err != nil {
				return err
			}

			return operations.Create(&operations.CreateOpts{
				ID:     containerID,
				Bundle: bundle,
			})
		},
	}

	cwd, _ := os.Getwd()
	cmd.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")

	return cmd
}
