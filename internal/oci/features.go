package oci

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/spf13/cobra"
)

func featuresCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "features",
		Short:   "List supported runtime features",
		Example: "  anocir features",
		RunE: func(cmd *cobra.Command, args []string) error {
			features, err := json.Marshal(container.GetFeatures())
			if err != nil {
				return fmt.Errorf("failed to get features: %w", err)
			}

			formattedFeatures, err := formatFeaturesOutput(features)
			if err != nil {
				return fmt.Errorf("failed to format output: %w", err)
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), formattedFeatures); err != nil {
				return fmt.Errorf("failed to print features to stdout: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func formatFeaturesOutput(features []byte) (string, error) {
	var formattedFeatures bytes.Buffer
	if err := json.Indent(
		&formattedFeatures,
		features,
		"",
		"  ",
	); err != nil {
		return "", fmt.Errorf("json indent features output: %w", err)
	}

	return formattedFeatures.String(), nil
}
