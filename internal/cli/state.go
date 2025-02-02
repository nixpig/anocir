// internal/cli/state.go

package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func stateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Query state of a container",
		Example: "  anocir state busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			state, err := operations.State(&operations.StateOpts{
				ID: containerID,
			})
			if err != nil {
				return err
			}

			if _, err := cmd.OutOrStdout().Write(
				[]byte(state),
			); err != nil {
				logrus.Errorf("state operation failed: %s", err)
				return fmt.Errorf("write state to stdout: %w", err)
			}

			return nil
		},
	}

	return cmd
}
