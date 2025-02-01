// internal/cli/root.go

package cli

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "anocir",
		Short:        "An experimental Linux container runtime.",
		Long:         "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
		Example:      "",
		Version:      "0.0.1",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logfile, _ := cmd.Flags().GetString("log")
			logfile = "/var/log/anocir/log.txt"
			if f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, os.ModeAppend); err != nil {
				fmt.Printf("Warning: failed to open log file %s. Logging to stderr.\n", logfile)
				logrus.SetOutput(os.Stderr)
			} else {
				logrus.SetOutput(f)
			}

			debug, _ := cmd.Flags().GetBool("debug")
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			logrus.SetFormatter(&logrus.TextFormatter{
				DisableColors: false,
				FullTimestamp: true,
			})
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

	// TODO: implement for Docker?
	cmd.PersistentFlags().BoolP("systemd-cgroup", "", false, "placeholder")
	cmd.PersistentFlags().StringP("root", "", "", "placeholder")
	cmd.PersistentFlags().StringP("log-format", "", "", "placeholder")
	cmd.PersistentFlags().StringP(
		"log",
		"l",
		"/var/log/anocir/log.txt",
		"Location of log file",
	)
	cmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")

	cmd.CompletionOptions.HiddenDefaultCmd = true

	return cmd
}
