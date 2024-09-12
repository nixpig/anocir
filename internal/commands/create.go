package commands

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	cp "github.com/otiai10/copy"
)

func Create(containerID, bundlePath string) error {
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", containerID)

	if err := os.MkdirAll(containerPath, os.ModeDir); err != nil {
		if err == os.ErrExist {
			return pkg.ErrContainerExists
		}

		return fmt.Errorf("make brownie container directory: %w", err)
	}

	absBundlePath, err := filepath.Abs(bundlePath)
	if err != nil {
		return fmt.Errorf("get absolute path to bundle: %w", err)
	}

	configJson, err := os.ReadFile(filepath.Join(absBundlePath, "config.json"))
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJson, &spec); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	state := &specs.State{
		Version:     spec.Version,
		ID:          containerID,
		Status:      specs.StateCreating,
		Bundle:      absBundlePath,
		Annotations: spec.Annotations,
	}

	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save state (creating): %w", err)
	}

	bundleRootfs := filepath.Join(absBundlePath, spec.Root.Path)
	containerRootfs := filepath.Join(containerPath, spec.Root.Path)
	containerConfigPath := filepath.Join(containerPath, "config.json")

	if err := cp.Copy(bundleRootfs, containerRootfs); err != nil {
		return fmt.Errorf("copy bundle rootfs to container rootfs: %w", err)
	}

	if err := cp.Copy(filepath.Join(absBundlePath, "config.json"), containerConfigPath); err != nil {
		return fmt.Errorf("copy container spec: %w", err)
	}

	// 4. Invoke 'createRuntime' hooks.
	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(spec.Hooks.CreateRuntime); err != nil {
		return fmt.Errorf("run createRuntime hooks: %w", err)
	}

	// 5. Invoke 'createContainer' hooks.
	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(spec.Hooks.CreateContainer); err != nil {
		return fmt.Errorf("run createContainer hooks: %w", err)
	}

	initSockAddr := filepath.Join(containerPath, "init.sock")
	containerSockAddr := filepath.Join(containerPath, "container.sock")

	forkCmd := exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			containerID,
			initSockAddr,
			containerSockAddr,
		}...)

	var cloneFlags uintptr
	for _, ns := range spec.Linux.Namespaces {
		lns := internal.LinuxNamespace(ns)
		f, err := lns.ToFlag()
		if err != nil {
			return err
		}

		cloneFlags = cloneFlags | f
	}

	// apply configuration, e.g. devices, proc, etc...
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   cloneFlags,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      int(spec.Process.User.UID),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      int(spec.Process.User.GID),
				Size:        1,
			},
		},
	}

	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("fork create command: %w", err)
	}

	pid := forkCmd.Process.Pid
	if err := forkCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach from process: %w", err)
	}

	// wait for container to be ready
	if err := os.RemoveAll(initSockAddr); err != nil {
		return err
	}
	listener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on init socket: %w", err)
	}
	defer listener.Close()

	initConn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer initConn.Close()

	b := make([]byte, 128)

	for {
		n, err := initConn.Read(b)
		if err != nil || n == 0 {
			continue
		}

		if string(b[:n]) == "ready" {
			break
		}
	}

	state.Status = specs.StateCreated
	state.Pid = pid
	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
