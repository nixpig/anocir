package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	cp "github.com/otiai10/copy"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger) error {
	absBundlePath, err := filepath.Abs(opts.Bundle)
	if err != nil {
		return fmt.Errorf("absolute path to bundle: %w", err)
	}

	bundleSpecPath := filepath.Join(absBundlePath, "config.json")

	bundleSpecJSON, err := os.ReadFile(bundleSpecPath)
	if err != nil {
		return fmt.Errorf("read spec from bundle: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(bundleSpecJSON, &spec); err != nil {
		return fmt.Errorf("parse spec: %w", err)
	}

	if spec.Linux == nil {
		return errors.New("not a linux container")
	}

	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)

	if stat, _ := os.Stat(containerPath); stat != nil {
		return pkg.ErrContainerExists
	}

	if err := os.MkdirAll(containerPath, os.ModeDir); err != nil {
		return fmt.Errorf("make brownie container directory: %w", err)
	}

	state := &specs.State{
		Version:     spec.Version,
		ID:          opts.ID,
		Status:      specs.StateCreating,
		Bundle:      absBundlePath,
		Annotations: spec.Annotations,
	}

	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save creating state: %w", err)
	}

	bundleRootfs := filepath.Join(absBundlePath, spec.Root.Path)
	containerRootfs := filepath.Join(containerPath, spec.Root.Path)
	containerSpecPath := filepath.Join(containerPath, "config.json")

	if err := cp.Copy(bundleRootfs, containerRootfs); err != nil {
		return fmt.Errorf("copy bundle rootfs to container rootfs: %w", err)
	}

	if err := cp.Copy(bundleSpecPath, containerSpecPath); err != nil {
		return fmt.Errorf("copy bundle spec to container spec: %w", err)
	}

	if spec.Hooks != nil {
		// TODO: If error, destroy container and created resources then call 'poststop' hooks.
		if err := internal.ExecHooks(spec.Hooks.CreateRuntime); err != nil {
			return fmt.Errorf("execute createruntime hooks: %w", err)
		}

		// TODO: If error, destroy container and created resources then call 'poststop' hooks.
		if err := internal.ExecHooks(spec.Hooks.CreateContainer); err != nil {
			return fmt.Errorf("execute createcontainer hooks: %w", err)
		}
	}

	initSockAddr := filepath.Join(containerPath, "init.sock")
	containerSockAddr := filepath.Join(containerPath, "container.sock")

	if err := os.RemoveAll(initSockAddr); err != nil {
		return err
	}
	listener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on init socket: %w", err)
	}
	defer listener.Close()

	forkCmd := exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			opts.ID,
			initSockAddr,
			containerSockAddr,
		}...)

	var cloneFlags uintptr
	log.Info().Msg("convert namespaces to flags")
	for _, ns := range spec.Linux.Namespaces {
		ns := internal.LinuxNamespace(ns)
		flag, err := ns.ToFlag()
		if err != nil {
			log.Error().Err(err).Msg("convert namespace to flag")
			return fmt.Errorf("convert namespace to flag: %w", err)
		}

		cloneFlags = cloneFlags | flag
	}
	log.Info().Msg("set sysprocattr")
	// apply configuration, e.g. devices, proc, etc...
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		// TODO: presumably this should be clone flags from namespaces in the config spec??
		Cloneflags: cloneFlags,
		// Cloneflags: syscall.CLONE_NEWUTS |
		// 	syscall.CLONE_NEWPID |
		// 	syscall.CLONE_NEWUSER |
		// 	syscall.CLONE_NEWNET |
		// 	syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: int(spec.Process.User.UID),
				HostID:      os.Geteuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: int(spec.Process.User.GID),
				HostID:      os.Getegid(),
				Size:        1,
			},
		},
	}

	forkCmd.Env = spec.Process.Env

	log.Info().Msg("start fork cmd")
	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("fork: %w", err)
	}

	log.Info().Msg("release second fork process")
	if err := forkCmd.Process.Release(); err != nil {
		log.Error().Err(err).Msg("detach fork")
		return err
	}
	log.Info().Msg("process successfully released")

	initConn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer initConn.Close()

	b := make([]byte, 128)

	fmt.Println("listening on: ", initSockAddr)
	for {
		time.Sleep(time.Second)

		n, err := initConn.Read(b)
		if err != nil || n == 0 {
			if err == io.EOF {
				fmt.Println("error: received EOF from socket")
				os.Exit(1)
			}

			fmt.Println("error: ", err)
			continue
		}

		if len(b) >= 5 && string(b[:5]) == "ready" {
			fmt.Println("received 'ready' message")
			log.Info().Msg("received 'ready' message")
			break
		}
	}

	state.Status = specs.StateCreated
	pid := forkCmd.Process.Pid
	state.Pid = pid
	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
