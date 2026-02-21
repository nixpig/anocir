package oci

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func reexecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "reexec [flags] CONTAINER_ID",
		Short:  internalUseMessage,
		Args:   cobra.ExactArgs(1),
		Hidden: true, // this command is only used internally
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			consoleSocketFD, _ := cmd.Flags().GetInt("console-socket-fd")
			rootDir, _ := cmd.Flags().GetString("root")

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			cntr.ConsoleSocketFD = consoleSocketFD
			if err := cntr.Reexec(); err != nil {
				return fmt.Errorf("failed to reexec container: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntP("console-socket-fd", "", 0, "Console socket FD")

	return cmd
}
