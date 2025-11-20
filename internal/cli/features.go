package cli

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/nixpig/anocir/internal/operations"
	"github.com/spf13/cobra"
)

func featuresCmd() *cobra.Command {
	features := &cobra.Command{
		Use:     "features",
		Short:   "List supported runtime features",
		Example: "  anocir features",
		RunE: func(cmd *cobra.Command, args []string) error {
			features, err := json.Marshal(operations.GetFeatures())
			if err != nil {
				return fmt.Errorf("features: %w", err)
			}

			var formattedFeatures bytes.Buffer
			if err := json.Indent(
				&formattedFeatures,
				[]byte(features),
				"",
				"  ",
			); err != nil {
				return err
			}

			if _, err := cmd.OutOrStdout().Write(
				formattedFeatures.Bytes(),
			); err != nil {
				return fmt.Errorf("features: %w", err)
			}

			return nil
		},
	}

	return features
}
