package oci

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/nixpig/anocir/internal/container"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update [flags] CONTAINER_ID",
		Short:   "Update resource limits of existing container",
		Example: "  anocir update --resources resources.json busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			rootDir, _ := cmd.Flags().GetString("root")
			resources, _ := cmd.Flags().GetString("resources")

			var linuxResources specs.LinuxResources
			var data []byte
			var err error

			switch resources {
			case "-":
				data, err = io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf(
						"failed to read resources from stdin: %w",
						err,
					)
				}
			case "":
				// TODO: Parse flags.
				return fmt.Errorf(
					"resources can currently only be read from a resources.json file or passed via stdin",
				)
			default:
				data, err = os.ReadFile(resources)
				if err != nil {
					return fmt.Errorf("failed to load resources file: %w", err)
				}
			}

			if err := json.Unmarshal(data, &linuxResources); err != nil {
				return fmt.Errorf("failed to parse resources JSON: %w", err)
			}

			cntr, err := container.Load(containerID, rootDir)
			if err != nil {
				return fmt.Errorf("failed to load container: %w", err)
			}

			state, err := cntr.GetState()
			if err != nil {
				return fmt.Errorf("failed to get container state: %w", err)
			}

			if err := platform.UpdateCgroup(
				cntr.GetSpec().Linux.CgroupsPath,
				state.ID,
				&linuxResources,
			); err != nil {
				return fmt.Errorf("failed to update cgroups: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().
		StringP("resources", "r", "resources.json", "path to resources JSON file, pass \"-\" to read from stdin")

	return cmd
}
