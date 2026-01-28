package oci

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func killCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kill [flags] CONTAINER_ID SIGNAL",
		Short:   "Send a signal to a container",
		Example: "  anocir kill busybox 9",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			signal := args[1]

			rootDir, _ := cmd.Flags().GetString("root")
			killAll, _ := cmd.Flags().GetBool("all")

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			return cntr.DoWithLock(func(c *container.Container) error {
				if err := c.Kill(signal, killAll); err != nil {
					return fmt.Errorf("failed to kill container: %w", err)
				}
				return nil
			})
		},
	}

	cmd.Flags().
		BoolP("all", "a", false, "send signal to all proceses in container cgroup")

	return cmd
}
