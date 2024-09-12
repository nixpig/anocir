package cmd

import (
	"github.com/nixpig/brownie/internal/commands"
	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:     "brownie",
	Short:   "An experimental Linux container runtime.",
	Long:    "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
	Example: "",
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

var Create = &cobra.Command{
	Use:     "create [flags] CONTAINER_ID PATH_TO_BUNDLE",
	Short:   "Create a container",
	Args:    cobra.ExactArgs(2),
	Example: "  brownie create busybox ./busybox",
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		bundlePath := args[1]

		return commands.Create(containerID, bundlePath)
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

var Fork = &cobra.Command{
	Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
	Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
	Args:    cobra.ExactArgs(3),
	Example: "\n -- FOR INTERNAL USE ONLY --",
	Hidden:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		initSockAddr := args[1]
		containerSockAddr := args[2]

		return commands.Fork(containerID, initSockAddr, containerSockAddr)
	},
}

func init() {
	Root.AddCommand(
		Create,
		Start,
		QueryState,
		Delete,
		Kill,
		Fork,
	)

	Root.CompletionOptions.HiddenDefaultCmd = true
}
