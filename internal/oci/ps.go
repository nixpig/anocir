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
		Short:   "Display the running processes in a container",
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

			state, err := cntr.GetState()
			if err != nil {
				return fmt.Errorf("failed to get container state: %w", err)
			}

			processes, err := platform.GetCgroupProcesses(
				cntr.GetSpec().Linux.CgroupsPath,
				state.ID,
			)
			if err != nil {
				return fmt.Errorf("failed to get processes: %w", err)
			}

			formattedOutput, err := formatProcessesOutput(format, processes)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), formattedOutput); err != nil {
				return fmt.Errorf("failed to print processes: %w", err)
			}

			return nil
		},
	}

	// TODO: Implement.
	cmd.Flags().StringP("format", "f", "table", "format for ps output")

	return cmd
}

func formatProcessesOutput(format string, processes []int) (string, error) {
	switch format {
	case "table":
		var b strings.Builder
		for _, p := range processes {
			fmt.Fprintf(&b, "%d\n", p)
		}
		return b.String(), nil
	case "json":
		data, err := json.Marshal(processes)
		if err != nil {
			return "", fmt.Errorf("create processes output json: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("invalid format: %s", format)
	}
}
