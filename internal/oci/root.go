package oci

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/logging"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "anocir",
		Short:   "An experimental Linux container runtime",
		Long:    "An experimental Linux container runtime, implementing the OCI Runtime Spec",
		Example: "",
		// TODO: Bake version in at build time.
		Version:      "0.0.1",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !platform.IsUnifiedCgroupsMode() {
				return errors.New("anocir requires cgroup v2 (unified mode)")
			}

			logfile, _ := cmd.Flags().GetString("log")
			debug, _ := cmd.Flags().GetBool("debug")
			logFormat, _ := cmd.Flags().GetString("log-format")

			if logfile != "" {
				// TODO: Tidy all this logic up. Do we really want to not log?
				if err := os.MkdirAll(filepath.Dir(logfile), 0o755); err != nil {
					fmt.Fprintf(
						cmd.ErrOrStderr(),
						"Warning: failed to create log directory: %s",
						err,
					)
				} else {
					f, err := os.OpenFile(
						logfile,
						os.O_CREATE|os.O_APPEND|os.O_WRONLY,
						0o644,
					)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to open log file '%s': %s", logfile, err)
					} else {
						logger := logging.NewLogger(f, debug, logFormat)
						slog.SetDefault(logger)
					}
				}
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
		execCmd(),
		childExecCmd(),
		psCmd(),
		updateCmd(),
		pauseCmd(),
		resumeCmd(),
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

	cmd.PersistentFlags().Bool("debug", false, "Enable debug logging")

	cmd.CompletionOptions.HiddenDefaultCmd = true

	// Flags required by Docker.
	// systemd is always used, cgroup is cgroupPath from spec or {containerID}.slice
	cmd.PersistentFlags().BoolP("systemd-cgroup", "", false, "Not implemented")
	cmd.PersistentFlags().StringP("log-format", "", "", "Specify log format")

	// ---

	return cmd
}
