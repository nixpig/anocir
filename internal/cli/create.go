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

			bundle, _ := cmd.Flags().GetString("bundle")

			consoleSocket, _ := cmd.Flags().GetString("console-socket")

			pidFile, _ := cmd.Flags().GetString("pid-file")

			rootDir, _ := cmd.Flags().GetString("root")

			logFile, _ := cmd.Root().Flags().GetString("log")

			if err := operations.Create(&operations.CreateOpts{
				ID:            containerID,
				Bundle:        bundle,
				ConsoleSocket: consoleSocket,
				PIDFile:       pidFile,
				RootDir:       rootDir,
				LogFile:       logFile,
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
