package cli

import (
	"fmt"

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

			if logfile != "" {
				logger, err := logging.NewLogger(logfile, debug)
				if err != nil {
					return fmt.Errorf("initialise logging: %w", err)
				}

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
	)

	// Required by Docker
	cmd.PersistentFlags().BoolP("systemd-cgroup", "", false, "Not implemented")
	cmd.PersistentFlags().StringP("log-format", "", "", "Not implemented")

	hideFlags(cmd, []string{"systemd-cgroup", "log-format"})

	// ---

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
		"Destination to write error logs (default is stderr)",
	)

	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")

	cmd.CompletionOptions.HiddenDefaultCmd = true

	return cmd
}

func hideFlags(cmd *cobra.Command, flags []string) {
	helpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, s []string) {
		for _, flag := range flags {
			cmd.Flags().MarkHidden(flag)
		}

		helpFunc(cmd, s)
	})
}
