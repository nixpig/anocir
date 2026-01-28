package oci

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nixpig/anocir/internal/container"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/spf13/cobra"
)

func psCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ps [flags] CONTAINER_ID [ps_args]",
		Short:   "Display the processes inside a container",
		Example: "  anocir ps busybox",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, _ := cmd.Flags().GetString("root")
			format, _ := cmd.Flags().GetString("format")

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			return cntr.DoWithLock(func(c *container.Container) error {
				state, err := c.GetState()
				if err != nil {
					return fmt.Errorf("failed to get container state: %w", err)
				}

				processes, err := platform.GetCgroupProcesses(
					c.GetSpec().Linux.CgroupsPath,
					state.ID,
				)
				if err != nil {
					return fmt.Errorf("failed to get processes: %w", err)
				}

				var output string

				switch format {
				case "table":
					var b strings.Builder
					for _, p := range processes {
						b.WriteString(fmt.Sprintf("%d", p))
					}
					output = b.String()
				case "json":
					data, err := json.Marshal(processes)
					if err != nil {
						return fmt.Errorf("failed to create json: %w", err)
					}
					output = string(data)
				default:
					return fmt.Errorf("invalid format: %s", format)
				}

				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", output); err != nil {
					return fmt.Errorf("failed to print processes: %w", err)
				}

				return nil
			})
		},
	}

	// TODO: Implement.
	cmd.Flags().StringP("format", "f", "table", "Format for ps output")

	return cmd
}
