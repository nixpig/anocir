package oci

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/logging"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "anocir",
		Short:        "An experimental Linux container runtime",
		Long:         "An experimental Linux container runtime, implementing the OCI Runtime Spec",
		Example:      "",
		Version:      "0.0.1",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logfile, _ := cmd.Flags().GetString("log")
			debug, _ := cmd.Flags().GetBool("debug")
			logFormat, _ := cmd.Flags().GetString("log-format")

			if logfile != "" {
				if err := os.MkdirAll(filepath.Dir(logfile), 0o755); err != nil {
					return fmt.Errorf("create log directory: %w", err)
				}

				f, err := os.OpenFile(
					logfile,
					os.O_CREATE|os.O_APPEND|os.O_WRONLY,
					0o644,
				)
				if err != nil {
					return fmt.Errorf("open log file %s: %w", logfile, err)
				}

				logger := logging.NewLogger(f, debug, logFormat)

				cmd.Root().SetErr(logging.NewErrorWriter(logger))
			}

			return nil
		},
	}

	cmd.AddCommand(
		stateCmd(),
		createCmd(),
		startCmd(),
		deleteCmd(),
		killCmd(),
		reexecCmd(),
		featuresCmd(),
		listCmd(),
	)

	cmd.PersistentFlags().StringP(
		"root",
		"",
		"/run/anocir",
		"Root directory for container state",
	)

	cmd.PersistentFlags().StringP(
		"log",
		"l",
		"",
		"Destination to write error logs (default \"stderr\")",
	)

	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")

	cmd.CompletionOptions.HiddenDefaultCmd = true

	// Flags required by Docker.
	// systemd is always used, cgroup is cgroupPath from spec or {containerID}.slice
	cmd.PersistentFlags().BoolP("systemd-cgroup", "", false, "Not implemented")
	cmd.PersistentFlags().StringP("log-format", "", "", "Not implemented")
	// ---

	return cmd
}
