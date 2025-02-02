package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start [flags] CONTAINER_ID",
		Short:   "Start a container",
		Example: "  anocir start busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			if err := operations.Start(&operations.StartOpts{
				ID: containerID,
			}); err != nil {
				logrus.Errorf("start operation failed: %s", err)
				return fmt.Errorf("start: %w", err)
			}

			return nil
		},
	}

	return cmd
}
