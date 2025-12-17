package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func reexecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "reexec [flags] CONTAINER_ID",
		Short:  "\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:   cobra.ExactArgs(1),
		Hidden: true, // this command is only used internally
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			consoleSocketFD, _ := cmd.Flags().GetInt("console-socket-fd")
			rootDir, _ := cmd.Flags().GetString("root")

			if err := operations.Reexec(&operations.ReexecOpts{
				ID:              containerID,
				ConsoleSocketFD: consoleSocketFD,
				RootDir:         rootDir,
			}); err != nil {
				return fmt.Errorf("failed to reexec process: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntP("console-socket-fd", "", 0, "Console socket FD")

	return cmd
}
