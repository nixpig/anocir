package cmd

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nixpig/brownie/container"
	"github.com/nixpig/brownie/internal/commands"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func RootCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
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
		filepath.Join(pkg.BrownieRootDir, "logs", "brownie.log"),
		"Location of log file",
	)

	return root
}

func createCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
	create := &cobra.Command{
		Use:     "create [flags] CONTAINER_ID",
		Short:   "Create a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie create busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg(" >> CREATE << ")
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

			return commands.Create(opts, log, db)
		},
	}

	cwd, _ := os.Getwd()
	create.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")
	create.Flags().StringP("console-socket", "s", "", "Console socket")
	create.Flags().StringP("pid-file", "p", "", "File to write container PID to")

	return create
}

func startCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
	start := &cobra.Command{
		Use:     "start [flags] CONTAINER_ID",
		Short:   "Start a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie start busybox",
	}

	start.RunE = func(cmd *cobra.Command, args []string) error {
		log.Info().Msg(" >> START << ")
		containerID := args[0]

		opts := &container.StartOpts{
			ID: containerID,
		}

		if err := commands.Start(opts, log, db); err != nil {
			return fmt.Errorf("fucked trying to start: %w", err)
		}

		return nil
	}

	return start
}

func killCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
	kill := &cobra.Command{
		Use:     "kill [flags] CONTAINER_ID SIGNAL",
		Short:   "Kill a container",
		Args:    cobra.ExactArgs(2),
		Example: "  brownie kill busybox 9",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg(" >> KILL << ")
			containerID := args[0]
			signal := args[1]

			opts := &container.KillOpts{
				ID:     containerID,
				Signal: signal,
			}

			return commands.Kill(opts, log, db)
		},
	}

	return kill
}

func deleteCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
	del := &cobra.Command{
		Use:     "delete [flags] CONTAINER_ID",
		Short:   "Delete a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie delete busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg(" >> DELETE << ")
			containerID := args[0]

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			opts := &container.DeleteOpts{
				ID:    containerID,
				Force: force,
			}

			return commands.Delete(opts, log, db)
		},
	}

	del.Flags().BoolP("force", "f", false, "force delete")

	return del
}

func forkCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
	fork := &cobra.Command{
		Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
		Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:    cobra.ExactArgs(3),
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg(" >> FORK << ")
			containerID := args[0]
			initSockAddr := args[1]

			consoleSocketFD, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("convert console socket fd to int: %w", err)
			}

			opts := &container.ForkOpts{
				ID:              containerID,
				InitSockAddr:    initSockAddr,
				ConsoleSocketFD: consoleSocketFD,
			}

			log.Info().Msg("loading container")
			cntr, err := container.Load(opts.ID, log, db)
			if err != nil {
				return err
			}

			log.Info().Msg("forking container")
			if err := cntr.Fork(opts, log, db); err != nil {
				log.Error().Err(err).Msg("failed to fork container")
				cntr.State.Status = specs.StateStopped
				if err := cntr.SaveState(); err != nil {
					log.Error().Err(err).Msg("failed to write state file")
					return fmt.Errorf("write state file: %w", err)
				}
			}

			return nil
		},
	}

	return fork
}

func stateCmd(log *zerolog.Logger, db *sql.DB) *cobra.Command {
	state := &cobra.Command{
		Use:     "state [flags] CONTAINER_ID",
		Short:   "Query a container state",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie state busybox",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg(" >> STATE << ")
			containerID := args[0]

			opts := &commands.StateOpts{
				ID: containerID,
			}

			state, err := commands.State(opts, log, db)
			if err != nil {
				return err
			}

			var formattedState bytes.Buffer
			if err := json.Indent(&formattedState, []byte(state), "", "  "); err != nil {
				log.Error().Err(err).Msg("failed to format state as json")
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
