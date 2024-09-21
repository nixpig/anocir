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
	"strconv"
	"syscall"
	"time"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/internal/capabilities"
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
			strconv.Itoa(state.Pid),
		}...)

	var cloneFlags uintptr
	for _, ns := range spec.Linux.Namespaces {
		ns := internal.LinuxNamespace(ns)
		flag, err := ns.ToFlag()
		if err != nil {
			log.Error().Err(err).Msg("convert namespace to flag")
			return fmt.Errorf("convert namespace to flag: %w", err)
		}

		cloneFlags = cloneFlags | flag
	}

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap
	if spec.Process != nil {
		cloneFlags = cloneFlags | syscall.CLONE_NEWUSER

		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: int(spec.Process.User.UID),
			HostID:      os.Geteuid(),
			Size:        1,
		})

		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: int(spec.Process.User.GID),
			HostID:      os.Getegid(),
			Size:        1,
		})
	}

	for _, uidMapping := range spec.Linux.UIDMappings {
		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: int(uidMapping.ContainerID),
			HostID:      int(uidMapping.HostID),
			Size:        int(uidMapping.Size),
		})
	}

	for _, gidMapping := range spec.Linux.GIDMappings {
		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: int(gidMapping.ContainerID),
			HostID:      int(gidMapping.HostID),
			Size:        int(gidMapping.Size),
		})
	}

	var ambientCapsFlags []uintptr
	for _, cap := range spec.Process.Capabilities.Ambient {
		ambientCapsFlags = append(ambientCapsFlags, uintptr(capabilities.Capabilities[cap]))
	}

	// apply configuration, e.g. devices, proc, etc...
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                ambientCapsFlags,
		Cloneflags:                 cloneFlags,
		Unshareflags:               syscall.CLONE_NEWNS,
		GidMappingsEnableSetgroups: false,
		UidMappings:                uidMappings,
		GidMappings:                gidMappings,
	}

	allowed := []string{
		// allow mknod for any device
		"c *:* m",
		"b *:* m",

		// /dev/null, zero, full
		"c 1:3 rwm",
		"c 1:5 rwm",
		"c 1:7 rwm",

		// consoles
		"c 5:1 rwm",
		"c 5:0 rwm",
		"c 4:0 rwm",
		"c 4:1 rwm",

		// /dev/urandom,/dev/random
		"c 1:9 rwm",
		"c 1:8 rwm",

		// /dev/pts/ - pts namespaces are "coming soon"
		"c 136:* rwm",
		"c 5:2 rwm",

		// tuntap
		"c 10:200 rwm",
	}
	cgroups := filepath.Join(containerRootfs, "sys/fs/cgroup/1")
	os.MkdirAll(cgroups, 0755)
	for _, val := range allowed {
		f, err := os.OpenFile(filepath.Join(cgroups, "devices.allow"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.Write(
			[]byte(fmt.Sprintf("%s\n", val)),
		); err != nil {
			return fmt.Errorf("create devices cgroups: %w", err)
		}
	}

	forkCmd.Env = spec.Process.Env

	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("fork: %w", err)
	}

	// need to get the pid off the process _before_ releasing it
	state.Pid = forkCmd.Process.Pid
	if err := forkCmd.Process.Release(); err != nil {
		log.Error().Err(err).Msg("detach fork")
		return err
	}

	initConn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer initConn.Close()

	b := make([]byte, 128)

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
			log.Info().Msg("ready")
			break
		}
	}

	state.Status = specs.StateCreated
	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
