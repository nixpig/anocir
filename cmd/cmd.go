package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nixpig/brownie/internal/commands"
	"github.com/nixpig/brownie/pkg"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "brownie",
		Short:   "An experimental Linux container runtime.",
		Long:    "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
		Example: "",
		Version: "0.0.1",
	}

	root.AddCommand(
		createCmd(),
		startCmd(),
		stateCmd(),
		deleteCmd(),
		killCmd(),
		forkCmd(),
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

func createCmd() *cobra.Command {
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

			// TODO: set logging level
			log, err := createLogger(cmd)
			if err != nil {
				return err
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

func startCmd() *cobra.Command {
	start := &cobra.Command{
		Use:     "start [flags] CONTAINER_ID",
		Short:   "Start a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie start busybox",
	}

	start.RunE = func(cmd *cobra.Command, args []string) error {
		containerID := args[0]

		opts := &commands.StartOpts{
			ID: containerID,
		}

		log, err := createLogger(cmd)
		if err != nil {
			return err
		}

		return commands.Start(opts, log)
	}

	return start
}

func killCmd() *cobra.Command {
	kill := &cobra.Command{
		Use:     "kill [flags] CONTAINER_ID SIGNAL",
		Short:   "Kill a container",
		Args:    cobra.ExactArgs(2),
		Example: "  brownie kill busybox 9",
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			signal := args[1]

			opts := &commands.KillOpts{
				ID:     containerID,
				Signal: signal,
			}

			log, err := createLogger(cmd)
			if err != nil {
				return err
			}

			return commands.Kill(opts, log)
		},
	}

	return kill
}

func deleteCmd() *cobra.Command {
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

			opts := &commands.DeleteOpts{
				ID:    containerID,
				Force: force,
			}

			log, err := createLogger(cmd)
			if err != nil {
				return err
			}

			return commands.Delete(opts, log)
		},
	}

	del.Flags().BoolP("force", "f", false, "force delete")

	return del
}

func forkCmd() *cobra.Command {
	fork := &cobra.Command{
		Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
		Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:    cobra.ExactArgs(3),
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			initSockAddr := args[1]

			consoleSocketFD, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("convert console socket fd to int: %w", err)
			}

			opts := &commands.ForkOpts{
				ID:              containerID,
				InitSockAddr:    initSockAddr,
				ConsoleSocketFD: consoleSocketFD,
			}

			log, err := createLogger(cmd)
			if err != nil {
				return err
			}

			return commands.Fork(opts, log)
		},
	}

	return fork
}

func stateCmd() *cobra.Command {
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

			log, err := createLogger(cmd)
			if err != nil {
				return err
			}

			state, err := commands.State(opts, log)
			if err != nil {
				return err
			}

			var formattedState bytes.Buffer
			json.Indent(&formattedState, []byte(state), "", "  ")

			log.Info().Str("state", formattedState.String()).Msg("formatted state")

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

func createLogger(cmd *cobra.Command) (*zerolog.Logger, error) {
	logPath, err := cmd.InheritedFlags().GetString("log")
	if err != nil {
		return nil, err
	}

	logDir, _ := filepath.Split(logPath)

	if err := os.MkdirAll(logDir, 0666); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logFile, err := os.OpenFile(
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	log := zerolog.New(logFile).With().Timestamp().Logger().Level(zerolog.InfoLevel)

	return &log, nil
}
