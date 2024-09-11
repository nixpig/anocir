package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
	"github.com/nixpig/brownie/pkg/config"
	cp "github.com/otiai10/copy"
)

const BrownieRootDir = "/var/lib/brownie"

func Create(containerID, bundlePath string) error {
	// 2. TODO: Create the container runtime environment according to configuration
	// in config.json.
	containerPath := filepath.Join(BrownieRootDir, "containers", containerID)

	if fi, err := os.Stat(containerPath); err == nil || fi != nil {
		return errors.New("container with specified ID already exists")
	}

	if err := os.MkdirAll(containerPath, os.ModeDir); err != nil {
		return fmt.Errorf("make brownie container directory: %w", err)
	}

	absBundlePath, err := filepath.Abs(bundlePath)
	if err != nil {
		return fmt.Errorf("get absolute path to bundle: %w", err)
	}

	c, err := os.ReadFile(filepath.Join(absBundlePath, "config.json"))
	if err != nil {
		return fmt.Errorf("read config.json: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(c, &cfg); err != nil {
		return fmt.Errorf("unmarshall config.json data: %w", err)
	}

	state := &pkg.State{
		OCIVersion:  cfg.OCIVersion,
		ID:          containerID,
		Status:      pkg.Creating,
		Bundle:      absBundlePath,
		Annotations: map[string]string{},
	}

	if err := saveState(state); err != nil {
		return fmt.Errorf("save creating state: %w", err)
	}

	bundleRootfs := filepath.Join(absBundlePath, cfg.Root.Path)
	containerRootfs := filepath.Join(containerPath, cfg.Root.Path)
	containerConfigPath := filepath.Join(containerPath, "config.json")

	if err := cp.Copy(bundleRootfs, containerRootfs); err != nil {
		return fmt.Errorf("copy bundle rootfs to container rootfs: %w", err)
	}

	if err := cp.Copy(filepath.Join(absBundlePath, "config.json"), containerConfigPath); err != nil {
		return fmt.Errorf("copy container config.json: %w", err)
	}

	if err := filesystem.MountProc(containerRootfs); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	if err := filesystem.MountDefaultDevices(containerRootfs); err != nil {
		return fmt.Errorf("mount dev: %w", err)
	}

	// 3. Invoke 'prestart' hooks.
	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(cfg.Hooks.Prestart); err != nil {
		return fmt.Errorf("run prestart hooks: %w", err)
	}

	// 4. Invoke 'createRuntime' hooks.
	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(cfg.Hooks.CreateRuntime); err != nil {
		return fmt.Errorf("run createRuntime hooks: %w", err)
	}

	// 5. Invoke 'createContainer' hooks.
	// TODO: If error, destroy container and created resources then call 'poststop' hooks.
	if err := internal.ExecHooks(cfg.Hooks.CreateContainer); err != nil {
		return fmt.Errorf("run createContainer hooks: %w", err)
	}

	forkCmd := exec.Command("/proc/self/exe", []string{"fork", "create", containerID, bundlePath}...)

	cloneFlags, err := internal.NamespacesToFlag(cfg.Linux.Namespaces)
	if err != nil {
		return fmt.Errorf("convert namespaces to flag: %w", err)
	}

	// apply configuration, e.g. devices, proc, etc...
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   *cloneFlags,
		Unshareflags: syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      cfg.Process.User.UID,
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      cfg.Process.User.GID,
				Size:        1,
			},
		},
	}

	fmt.Println("starting fork")
	// debugging...
	forkCmd.Stdout = os.Stdout
	forkCmd.Stderr = os.Stderr
	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("fork create command: %w", err)
	}

	fmt.Println("releasing process")
	pid := forkCmd.Process.Pid
	fmt.Println("pid: ", pid)
	if err := forkCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach from process: %w", err)
	}

	state.Status = pkg.Created
	state.PID = &pid
	if err := saveState(state); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}

func saveState(state *pkg.State) error {
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(
		filepath.Join(BrownieRootDir, "containers", state.ID, "state.json"),
		b,
		0644,
	); err != nil {
		return fmt.Errorf("write state to file: %w", err)
	}

	return nil
}
