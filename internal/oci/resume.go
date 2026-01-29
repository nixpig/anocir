package oci

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func resumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resume [flags] CONTAINER_ID",
		Short:   "Resume a paused container",
		Example: "  anocir resume busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, _ := cmd.Flags().GetString("root")

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			if err := cntr.Resume(); err != nil {
				return fmt.Errorf("failed to resume container: %w", err)
			}

			return nil
		},
	}

	return cmd
}
