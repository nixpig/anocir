package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"

	"github.com/nixpig/brownie/internal/commands"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func RootCmd(log *zerolog.Logger, logfile string) *cobra.Command {
	root := &cobra.Command{
		Use:          "brownie",
		Short:        "An experimental Linux container runtime.",
		Long:         "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
		Example:      "",
		Version:      "0.0.1",
		SilenceUsage: true,
	}

	root.AddCommand(
		createCmd(log),
		startCmd(log),
		stateCmd(log),
		deleteCmd(log),
		killCmd(log),
		reexecCmd(log),
	)

	// TODO: implement these flags for Docker
	root.PersistentFlags().BoolP("systemd-cgroup", "", false, "placeholder")
	root.PersistentFlags().StringP("root", "", "", "placeholder")
	root.PersistentFlags().StringP("log-format", "", "", "placeholder")

	root.CompletionOptions.HiddenDefaultCmd = true

	root.PersistentFlags().StringP(
		"log",
		"l",
		logfile,
		"Location of log file",
	)

	return root
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

			return commands.Create(&commands.CreateOpts{
				ID:            containerID,
				Bundle:        bundle,
				ConsoleSocket: consoleSocket,
				PIDFile:       pidFile,
			}, log)
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
	}

	start.RunE = func(cmd *cobra.Command, args []string) error {
		containerID := args[0]

		return commands.Start(&commands.StartOpts{
			ID: containerID,
		}, log)
	}

	return start
}

func killCmd(log *zerolog.Logger) *cobra.Command {
	kill := &cobra.Command{
		Use:     "kill [flags] CONTAINER_ID SIGNAL",
		Short:   "Kill a container",
		Args:    cobra.ExactArgs(2),
		Example: "  brownie kill busybox 9",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			signal := args[1]

			return commands.Kill(&commands.KillOpts{
				ID:     containerID,
				Signal: signal,
			}, log)
		},
	}

	kill.Flags().BoolP("all", "a", false, "all (docker)??")

	return kill
}

func deleteCmd(log *zerolog.Logger) *cobra.Command {
	del := &cobra.Command{
		Use:     "delete [flags] CONTAINER_ID",
		Short:   "Delete a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie delete busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			return commands.Delete(&commands.DeleteOpts{
				ID:    containerID,
				Force: force,
			}, log)
		},
	}

	del.Flags().BoolP("force", "f", false, "force delete")

	return del
}

func reexecCmd(log *zerolog.Logger) *cobra.Command {
	reexec := &cobra.Command{
		Use:     "reexec [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
		Short:   "Reexec container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:    cobra.ExactArgs(1),
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			stage, err := cmd.Flags().GetInt("stage")
			if err != nil {
				return err
			}

			opts := commands.ReexecOpts{
				ID: containerID,
			}

			switch stage {
			case 1:
				return commands.Reexec1(&opts, log)
			case 2:
				return commands.Reexec2(&opts, log)
			}

			return errors.New("invalid stage provided")
		},
	}

	reexec.Flags().IntP("stage", "s", 0, "stage of reexec (1-2)")

	return reexec
}

func stateCmd(log *zerolog.Logger) *cobra.Command {
	state := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Query a container state",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie state busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			state, err := commands.State(&commands.StateOpts{
				ID: containerID,
			}, log)
			if err != nil {
				return err
			}

			var formattedState bytes.Buffer
			if err := json.Indent(&formattedState, []byte(state), "", "  "); err != nil {
				return err
			}

			if _, err := cmd.OutOrStdout().Write(
				formattedState.Bytes(),
			); err != nil {
				return err
			}

			return nil
		},
	}

	return state
}
