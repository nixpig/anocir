// internal/cli/state.go

package cli

import (
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func stateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "state [flags] CONTAINER_ID",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			state, err := operations.State(&operations.StateOpts{
				ID: containerID,
			})
			if err != nil {
				return err
			}

			// TODO: do something with 'state'
			fmt.Println(state)

			return nil
		},
	}

	return cmd
}
