package oci

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   "List all containers",
		Example: "  anocir list",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")

			containerDirs, err := os.ReadDir(rootDir)
			if err != nil {
				return fmt.Errorf("failed to read container directory: %w", err)
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

			fmt.Fprint(w, "ID\tPID\tSTATE\t\n")

			for _, d := range containerDirs {
				id := d.Name()

				cntr, err := container.Load(id, rootDir)
				if err != nil {
					return fmt.Errorf(
						"failed to load container %s: %w",
						id,
						err,
					)
				}

				state, err := cntr.GetState()
				if err != nil {
					return fmt.Errorf(
						"failed to get state for container %s: %w",
						id,
						err,
					)
				}

				fmt.Fprintf(
					w,
					"%s\t%d\t%s\t\n",
					state.ID,
					state.Pid,
					state.Status,
				)
			}

			if err := w.Flush(); err != nil {
				return fmt.Errorf("failed to print container details: %w", err)
			}

			return nil
		},
	}

	return cmd
}
