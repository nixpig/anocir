package oci

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/container"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/validation"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
)

func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create [flags] CONTAINER_ID",
		Short:   "Create a container",
		Example: `  anocir create busybox`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !platform.IsUnifiedCgroupsMode() {
				return errors.New(
					"anocir requires cgroup v2 (unified mode)",
				)
			}

			containerID := args[0]

			if err := validation.ContainerID(containerID); err != nil {
				return fmt.Errorf("failed validation: %w", err)
			}

			bundle, _ := cmd.Flags().GetString("bundle")
			pidFile, _ := cmd.Flags().GetString("pid-file")
			rootDir, _ := cmd.Flags().GetString("root")
			consoleSocket, _ := cmd.Flags().GetString("console-socket")
			debug, _ := cmd.PersistentFlags().GetBool("debug")
			logFile, _ := cmd.Flags().GetString("log")
			logFormat, _ := cmd.Flags().GetString("log-format")

			if container.Exists(containerID, rootDir) {
				return fmt.Errorf("container '%s' exists", containerID)
			}

			spec, err := getContainerSpec(bundle)
			if err != nil {
				return fmt.Errorf("failed to get container spec: %w", err)
			}

			if err := createContainerDirs(rootDir, containerID); err != nil {
				return fmt.Errorf("failed to create container dirs: %w", err)
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

			if err := cntr.Init(); err != nil {
				return fmt.Errorf("failed to initialise container: %w", err)
			}

			return nil
		},
	}

	cwd, _ := os.Getwd()
	cmd.Flags().StringP("bundle", "b", cwd, "path of bundle directory")
	cmd.Flags().String("console-socket", "", "console socket path")
	cmd.Flags().String("pid-file", "", "file to write container PID to")

	return cmd
}

func getContainerSpec(path string) (*specs.Spec, error) {
	bundlePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute bundle path: %w", err)
	}

	config, err := os.ReadFile(filepath.Join(bundlePath, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read bundle config: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(config, &spec); err != nil {
		return nil, fmt.Errorf("parse bundle config: %w", err)
	}

	return spec, nil
}

func createContainerDirs(rootDir, containerID string) error {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return fmt.Errorf("create root dir: %w", err)
	}

	containerDir := filepath.Join(rootDir, containerID)
	if err := os.Mkdir(containerDir, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("container dir exists: %s", containerDir)
		}
		return fmt.Errorf("create container dir: %w", err)
	}

	return nil
}
