package cli

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/nixpig/brownie/internal/commands"
	"github.com/nixpig/brownie/internal/database"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func RootCmd(log *zerolog.Logger, db *database.DB, logfile string) *cobra.Command {
	root := &cobra.Command{
		Use:          "brownie",
		Short:        "An experimental Linux container runtime.",
		Long:         "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
		Example:      "",
		Version:      "0.0.1",
		SilenceUsage: true,
	}

	root.AddCommand(
		createCmd(log, db),
		startCmd(log, db),
		stateCmd(log, db),
		deleteCmd(log, db),
		killCmd(log, db),
		forkCmd(log, db),
	)

	root.CompletionOptions.HiddenDefaultCmd = true

	root.PersistentFlags().StringP(
		"log",
		"l",
		logfile,
		"Location of log file",
	)

	return root
}

func createCmd(log *zerolog.Logger, db *database.DB) *cobra.Command {
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
			}, log, db)
		},
	}

	cwd, _ := os.Getwd()
	create.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")
	create.Flags().StringP("console-socket", "s", "", "Console socket")
	create.Flags().StringP("pid-file", "p", "", "File to write container PID to")

	return create
}

func startCmd(log *zerolog.Logger, db *database.DB) *cobra.Command {
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
		}, log, db)
	}

	return start
}

func killCmd(log *zerolog.Logger, db *database.DB) *cobra.Command {
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
			}, log, db)
		},
	}

	return kill
}

func deleteCmd(log *zerolog.Logger, db *database.DB) *cobra.Command {
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
			}, log, db)
		},
	}

	del.Flags().BoolP("force", "f", false, "force delete")

	return del
}

func forkCmd(log *zerolog.Logger, db *database.DB) *cobra.Command {
	fork := &cobra.Command{
		Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
		Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:    cobra.ExactArgs(1),
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			return commands.Fork(&commands.ForkOpts{
				ID: containerID,
			}, log, db)
		},
	}

	return fork
}

func stateCmd(log *zerolog.Logger, db *database.DB) *cobra.Command {
	state := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Query a container state",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie state busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			state, err := commands.State(&commands.StateOpts{
				ID: containerID,
			}, log, db)
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
