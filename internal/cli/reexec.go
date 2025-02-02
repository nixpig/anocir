package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func reexecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reexec [flags] CONTAINER_ID",
		Short:   "Reexec container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Args:    cobra.ExactArgs(1),
		Hidden:  true, // this command is only used internally
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			// TODO: figure out a cleaner way of passing the console socket fd
			var consoleSocketFD *int
			if cmd.Flags().Changed("console-socket-fd") {
				flag, _ := cmd.Flags().GetInt("console-socket-fd")
				consoleSocketFD = &flag
			}

			if err := operations.Reexec(&operations.ReexecOpts{
				ID:              containerID,
				ConsoleSocketFD: consoleSocketFD,
			}); err != nil {
				logrus.Errorf("reexec operation failed: %s", err)
				return fmt.Errorf("reexec: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntP("console-socket-fd", "", 0, "console socket fd")

	return cmd
}
