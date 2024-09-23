package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/nixpig/brownie/internal/commands"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func RootCmd(log *zerolog.Logger) *cobra.Command {
	root := &cobra.Command{
		Use:     "brownie",
		Short:   "An experimental Linux container runtime.",
		Long:    "An experimental Linux container runtime; working towards OCI Runtime Spec compliance.",
		Example: "",
		Version: "0.0.1",
	}

	root.AddCommand(
		createCmd(log, root.OutOrStdout()),
		startCmd(log, root.OutOrStdout()),
		stateCmd(log),
		deleteCmd(log),
		killCmd(log),
		forkCmd(log),
	)

	root.CompletionOptions.HiddenDefaultCmd = true

	return root
}

func createCmd(log *zerolog.Logger, stdout io.Writer) *cobra.Command {
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
				ID:                containerID,
				Bundle:            bundle,
				ConsoleSocketPath: consoleSocket,
				PIDFile:           pidFile,
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

func startCmd(log *zerolog.Logger, stdout io.Writer) *cobra.Command {
	start := &cobra.Command{
		Use:     "start [flags] CONTAINER_ID",
		Short:   "Start a container",
		Args:    cobra.ExactArgs(1),
		Example: "  brownie start busybox",
	}

	start.OutOrStdout().Write([]byte("\n >>> before\n"))
	start.RunE = func(cmd *cobra.Command, args []string) error {
		start.OutOrStdout().Write([]byte("\n >>> after\n"))
		os.WriteFile("/tmp/out.txt", []byte("even written??"), 0666)
		stdout.Write([]byte("\n >> from start inner\n"))
		os.Stdout.Write([]byte("\n >> from start inner\n"))
		cmd.OutOrStdout().Write([]byte(" \n >>> cmd in start\n"))
		containerID := args[0]

		opts := &commands.StartOpts{
			ID: containerID,
		}

		start.OutOrStdout().Write([]byte("\n\n >>>>>>>> \n\n"))
		return commands.Start(opts, log, start.OutOrStdout(), start.ErrOrStderr())
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

			return commands.Kill(containerID, signal)
		},
	}

	return kill
}

func deleteCmd(log *zerolog.Logger) *cobra.Command {
	delete := &cobra.Command{
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

			return commands.Delete(opts, log)
		},
	}

	delete.Flags().BoolP("force", "f", false, "force delete")

	return delete
}

func forkCmd(log *zerolog.Logger) *cobra.Command {
	fork := &cobra.Command{
		Use:     "fork [flags] CONTAINER_ID INIT_SOCK_ADDR CONTAINER_SOCK_ADDR",
		Short:   "Fork container process\n\n \033[31m ⚠ FOR INTERNAL USE ONLY - DO NOT RUN DIRECTLY ⚠ \033[0m",
		Args:    cobra.ExactArgs(5),
		Example: "\n -- FOR INTERNAL USE ONLY --",
		Hidden:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			initSockAddr := args[1]
			containerSockAddr := args[2]
			pid := args[3]
			ipid, err := strconv.Atoi(pid)
			if err != nil {
				return fmt.Errorf("convert pid string to int: %w", err)
			}
			consoleSocketFD := args[4]
			iconsolesocketfd, err := strconv.Atoi(consoleSocketFD)
			if err != nil {
				return fmt.Errorf("convert console socket fd to int: %w", err)
			}

			opts := &commands.ForkOpts{
				ID:                containerID,
				InitSockAddr:      initSockAddr,
				ContainerSockAddr: containerSockAddr,
				PID:               ipid,
				ConsoleSocketFD:   iconsolesocketfd,
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
