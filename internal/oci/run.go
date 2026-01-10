package oci

import (
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/container"
	"github.com/nixpig/anocir/internal/validation"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run [flags] CONTAINER_ID",
		Short:   "Run a container",
		Example: "  anocir run busybox",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]

			bundle, _ := cmd.Flags().GetString("bundle")
			pidFile, _ := cmd.Flags().GetString("pid-file")
			rootDir, _ := cmd.Flags().GetString("root")
			consoleSocket, _ := cmd.Flags().GetString("console-socket")
			debug, _ := cmd.PersistentFlags().GetBool("debug")
			logFile, _ := cmd.Flags().GetString("log")
			logFormat, _ := cmd.Flags().GetString("log-format")
			_, _ = cmd.Flags().GetBool("detach")

			if err := validation.ContainerID(containerID); err != nil {
				return fmt.Errorf("failed validation: %w", err)
			}

			if container.Exists(containerID, rootDir) {
				return fmt.Errorf("container '%s' exists", containerID)
			}

			spec, err := getContainerSpec(bundle)
			if err != nil {
				return fmt.Errorf("failed to get container spec: %w", err)
			}

			if err := createContainerDirs(rootDir, containerID); err != nil {
				return fmt.Errorf(
					"failed to make container directories: %w",
					err,
				)
			}

			cntr := container.New(&container.Opts{
				ID:            containerID,
				Bundle:        bundle,
				Spec:          spec,
				ConsoleSocket: consoleSocket,
				PIDFile:       pidFile,
				RootDir:       rootDir,
				LogFile:       logFile,
				LogFormat:     logFormat,
				Debug:         debug,
			})

			if err := cntr.Lock(); err != nil {
				// This should never happen on a brand new container.
				return fmt.Errorf("failed to get lock on container: %w", err)
			}
			defer cntr.Unlock()

			if err := cntr.Save(); err != nil {
				return fmt.Errorf("failed to save container state: %w", err)
			}

			if err := cntr.Init(); err != nil {
				return fmt.Errorf("failed to initialise container: %w", err)
			}

			if err := cntr.Start(); err != nil {
				return fmt.Errorf("failed to start container: %w", err)
			}

			return nil
		},
	}

	cwd, _ := os.Getwd()
	cmd.Flags().StringP("bundle", "b", cwd, "Path to bundle directory")
	cmd.Flags().String("console-socket", "", "Console socket path")
	cmd.Flags().String("pid-file", "", "File to write container PID to")
	cmd.Flags().BoolP("detach", "d", false, "Detach from container process")

	return cmd
}
