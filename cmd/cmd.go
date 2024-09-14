package cmd

import (
	"os"

	"github.com/nixpig/brownie/internal/commands"
	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:     "brownie",
	Short:   "An experimental Linux container runtime.",
	Long:    "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
	Example: "",
}

func createCmd() *cobra.Command {
	var create = &cobra.Command{
		Use:     "create [flags] CONTAINER_ID",
		Short:   "Create a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie create busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			bundle, err := cmd.Flags().GetString("bundle")
			if err != nil {
				return err
			}

			consoleSocket, err := cmd.Flags().GetString("console-socket")
			if err != nil {
				return err
			}

			pidFile, err := cmd.Flags().GetString("pid-file")
			if err != nil {
				return err
			}

			opts := &commands.CreateOpts{
				ID:            containerID,
				Bundle:        bundle,
				ConsoleSocket: consoleSocket,
				PIDFile:       pidFile,
			}

			return commands.Create(opts)
		},
	}

	cwd, _ := os.Getwd()
	create.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")
	create.Flags().StringP("console-socket", "s", "", "Console socket")
	create.Flags().StringP("pid-file", "p", "", "File to write container PID to")

	return create
}

var Start = &cobra.Command{
	Use:     "start [flags] CONTAINER_ID",
	Short:   "Start a container",
	Args:    cobra.ExactArgs(1),
	Example: "  brownie start busybox",
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]

		return commands.Start(containerID)
	},
}

var Kill = &cobra.Command{
	Use:     "kill [flags] CONTAINER_ID SIGNAL",
	Short:   "Kill a container",
	Args:    cobra.ExactArgs(2),
	Example: "  brownie delete busybox 9",
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		signal := args[1]

		return commands.Kill(containerID, signal)
	},
}

var Delete = &cobra.Command{
	Use:     "delete [flags] CONTAINER_ID",
	Short:   "Delete a container",
	Args:    cobra.ExactArgs(1),
	Example: "  brownie delete busybox",
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]

		return commands.Delete(containerID)
	},
}

var Fork = &cobra.Command{
	Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
	Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
	Args:    cobra.ExactArgs(3),
	Example: "\n -- FOR INTERNAL USE ONLY --",
	Hidden:  true,
	Run: func(cmd *cobra.Command, args []string) {
		containerID := args[0]
		initSockAddr := args[1]
		containerSockAddr := args[2]

		commands.Fork(containerID, initSockAddr, containerSockAddr)
	},
}

var QueryState = &cobra.Command{
	Use:     "state [flags] CONTAINER_ID",
	Short:   "Query a container state",
	Args:    cobra.ExactArgs(1),
	Example: "  brownie state busybox",
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]

		return commands.QueryState(containerID)
	},
}

func init() {
	Root.AddCommand(
		createCmd(),
		Start,
		QueryState,
		Delete,
		Kill,
		Fork,
	)

	Root.CompletionOptions.HiddenDefaultCmd = true
}
