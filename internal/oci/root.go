package oci

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/nixpig/anocir/internal/logging"
	"github.com/spf13/cobra"
)

var Version = "dev"

const defaultRootDir = "/run/anocir"

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "anocir",
		Short:        "An experimental Linux container runtime",
		Long:         "An experimental Linux container runtime, implementing the OCI Runtime Spec",
		Example:      "",
		Version:      Version,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logFile, _ := cmd.Flags().GetString("log")
			debug, _ := cmd.Flags().GetBool("debug")
			logFormat, _ := cmd.Flags().GetString("log-format")

			w := io.Discard
			if logFile != "" {
				f, err := logging.OpenLogFile(logFile)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to open log file '%s': %s", logFile, err)
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

	cmd.PersistentFlags().StringP("root", "", defaultRootDir, "root directory for container state")
	cmd.PersistentFlags().StringP("log", "l", "", "destination to write logs")
	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	cmd.PersistentFlags().StringP("log-format", "", "text", "log format (json | text)")

	// systemd is always used. Flag is unused but provided to satisfy Docker expectation.
	cmd.PersistentFlags().BoolP("systemd-cgroup", "", false, "not implemented")

	cmd.CompletionOptions.HiddenDefaultCmd = true

	return cmd
}
