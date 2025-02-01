// internal/cli/kill.go

package cli

import (
	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func killCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kill [flags] CONTAINER_ID SIGNAL",
		Short:   "Send signal to a container",
		Example: "  anocir kill busybox 9",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			signal := args[1]

			return operations.Kill(&operations.KillOpts{
				ID:     containerID,
				Signal: signal,
			})
		},
	}

	// TODO: do something with this
	cmd.Flags().BoolP("all", "a", false, "Kill all (Docker??)")

	return cmd
}
