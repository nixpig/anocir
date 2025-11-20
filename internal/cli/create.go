package cli

import (
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create [flags] CONTAINER_ID",
		Short:   "Create a container",
		Long:    "Create a container from a bundle directory",
		Example: "  anocir create busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			bundle, err := cmd.Flags().GetString("bundle")
			if err != nil {
				return err
			}

			consoleSocket, err := cmd.Flags().GetString("console-socket")
			if err != nil {
				return err
			}

			pidFile, err := cmd.Flags().GetString("pid-file")
			if err != nil {
				return err
			}

			rootDir, err := cmd.Flags().GetString("root")
			if err != nil {
				return err
			}

			if err := operations.Create(&operations.CreateOpts{
				ID:            containerID,
				Bundle:        bundle,
				ConsoleSocket: consoleSocket,
				PIDFile:       pidFile,
				RootDir:       rootDir,
			}); err != nil {
				return fmt.Errorf("create: %w", err)
			}

			return nil
		},
	}

	cwd, _ := os.Getwd()
	cmd.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")
	cmd.Flags().StringP("console-socket", "s", "", "Console socket path")
	cmd.Flags().StringP("pid-file", "p", "", "File to write container PID to")

	return cmd
}
