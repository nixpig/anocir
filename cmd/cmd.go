package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal/commands"
	"github.com/nixpig/brownie/pkg"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:     "brownie",
	Short:   "An experimental Linux container runtime.",
	Long:    "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
	Example: "",
	Version: "0.0.1",
}

func createCmd(log *zerolog.Logger) *cobra.Command {
	create := &cobra.Command{
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

			return commands.Create(opts, log)
		},
	}

	cwd, _ := os.Getwd()
	create.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")
	create.Flags().StringP("console-socket", "s", "", "Console socket")
	create.Flags().StringP("pid-file", "p", "", "File to write container PID to")

	return create
}

func startCmd(log *zerolog.Logger) *cobra.Command {
	start := &cobra.Command{
		Use:     "start [flags] CONTAINER_ID",
		Short:   "Start a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie start busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			opts := &commands.StartOpts{
				ID: containerID,
			}

			return commands.Start(opts, log)
		},
	}

	return start
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

func forkCmd(log *zerolog.Logger) *cobra.Command {
	fork := &cobra.Command{
		Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
		Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:    cobra.ExactArgs(4),
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			stage := args[0]
			containerID := args[1]
			initSockAddr := args[2]
			containerSockAddr := args[3]

			if commands.ForkStage(stage) != commands.ForkIntermediate && commands.ForkStage(stage) != commands.ForkDetached {
				return errors.New("invalidate fork stage")
			}

			opts := &commands.ForkOpts{
				ID:                containerID,
				InitSockAddr:      initSockAddr,
				ContainerSockAddr: containerSockAddr,
				Stage:             commands.ForkStage(stage),
			}

			return commands.Fork(opts, log)
		},
	}

	return fork
}

func stateCmd(log *zerolog.Logger) *cobra.Command {
	state := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Query a container state",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie state busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			opts := &commands.StateOpts{
				ID: containerID,
			}

			state, err := commands.State(opts, log)
			if err != nil {
				e := cmd.ErrOrStderr()
				e.Write([]byte(err.Error()))
				return err
			}

			var prettified bytes.Buffer
			json.Indent(&prettified, []byte(state), "", "  ")

			fmt.Fprint(cmd.OutOrStdout(), prettified.String())
			return nil
		},
	}

	return state
}

func init() {
	logfile, err := os.OpenFile(
		filepath.Join(pkg.BrownieRootDir, "logs", "brownie.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		fmt.Println("open log file: %w", err)
		os.Exit(1)
	}

	log := zerolog.New(logfile).With().Timestamp().Logger()

	Root.AddCommand(
		createCmd(&log),
		startCmd(&log),
		stateCmd(&log),
		Delete,
		Kill,
		forkCmd(&log),
	)

	Root.CompletionOptions.HiddenDefaultCmd = true
}
