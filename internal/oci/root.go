package oci

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/nixpig/anocir/internal/logging"
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
			logFile, _ := cmd.Flags().GetString("log")
			debug, _ := cmd.Flags().GetBool("debug")
			logFormat, _ := cmd.Flags().GetString("log-format")

			w := io.Discard
			if logFile != "" {
				f, err := logging.OpenLogFile(logFile)
				if err != nil {
					fmt.Fprintf(
						cmd.ErrOrStderr(),
						"Warning: failed to open log file '%s': %s",
						logFile,
						err,
					)
				} else {
					w = f
				}
			}

			slog.SetDefault(logging.NewLogger(w, debug, logFormat))

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
		"root directory for container state",
	)

	cmd.PersistentFlags().StringP(
		"log",
		"l",
		"",
		"destination to write logs",
	)

	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")

	cmd.CompletionOptions.HiddenDefaultCmd = true

	// Flags required by Docker.
	// systemd is always used, cgroup is cgroupPath from spec or {containerID}.slice
	// TODO: Review implementation for this.
	cmd.PersistentFlags().BoolP("systemd-cgroup", "", false, "not implemented")
	cmd.PersistentFlags().
		StringP("log-format", "", "text", "log format (json | text)")

	// ---

	return cmd
}
