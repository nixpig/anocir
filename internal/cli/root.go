// internal/cli/root.go

package cli

import "github.com/spf13/cobra"

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "anocir",
		SilenceUsage: true,
	}

	cmd.AddCommand(
		stateCmd(),
		createCmd(),
		startCmd(),
		deleteCmd(),
		killCmd(),
		reexecCmd(),
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

	cmd.CompletionOptions.HiddenDefaultCmd = true

	return cmd
}
