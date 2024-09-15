package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func server(conn net.Conn, spec specs.Spec) {
	defer conn.Close()

	b := make([]byte, 128)

	for {
		n, err := conn.Read(b)
		if err != nil {
			// TODO: log it
			fmt.Println(fmt.Errorf("read from connection: %s", err))
		}

		if n == 0 {
			break
		}

		switch string(b[:n]) {
		case "start":
			cmd := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
			cmd.Stdout = conn
			cmd.Stderr = conn
			if err := cmd.Run(); err != nil {
				conn.Write([]byte(err.Error()))
			}
			return
		}
	}
}

type ForkStage string

var (
	ForkIntermediate ForkStage = "intermediate"
	ForkDetached     ForkStage = "detached"
)

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ContainerSockAddr string
	Stage             ForkStage
}

func Fork(opts *ForkOpts, log *zerolog.Logger) error {
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)
	configJSON, err := os.ReadFile(filepath.Join(containerPath, "config.json"))
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	switch opts.Stage {
	case ForkIntermediate:
		if spec.Linux == nil {
			return errors.New("not a linux container")
		}
		var cloneFlags uintptr
		for _, ns := range spec.Linux.Namespaces {
			ns := internal.LinuxNamespace(ns)
			flag, err := ns.ToFlag()
			if err != nil {
				return fmt.Errorf("convert namespace to flag: %w", err)
			}

			cloneFlags = cloneFlags | flag
		}

		initSockAddr := filepath.Join(containerPath, "init.sock")
		containerSockAddr := filepath.Join(containerPath, "container.sock")

		forkCmd := exec.Command(
			"/proc/self/exe",
			[]string{
				"fork",
				string(ForkDetached),
				opts.ID,
				initSockAddr,
				containerSockAddr,
			}...)

		// apply configuration, e.g. devices, proc, etc...
		forkCmd.SysProcAttr = &syscall.SysProcAttr{
			// Cloneflags:   cloneFlags,
			Cloneflags: syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWUSER |
				syscall.CLONE_NEWNET |
				syscall.CLONE_NEWNS,
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

		log.Info().Msg("before fork start")
		if err := forkCmd.Start(); err != nil {
			log.Error().Err(err).Msg("error in fork start!")
			return fmt.Errorf("re-fork: %w", err)
		}
		log.Info().Msg("after fork start")

		// pid := forkCmd.Process.Pid
		if err := forkCmd.Process.Release(); err != nil {
			log.Error().Err(err).Msg("failed to detach")
			return fmt.Errorf("detach refork: %w", err)
		}

	case ForkDetached:
		initConn, err := net.Dial("unix", opts.InitSockAddr)
		if err != nil {
			// TODO: log it
			fmt.Println(fmt.Errorf("dialing init socket: %s", err))
			return err
		}
		defer initConn.Close()

		if err := os.RemoveAll(opts.ContainerSockAddr); err != nil {
			initConn.Write([]byte(fmt.Sprintf("remove socket: %s", err)))
			return err
		}

		listener, err := net.Listen("unix", opts.ContainerSockAddr)
		if err != nil {
			initConn.Write([]byte(fmt.Sprintf("listen on socket: %s", err)))
			return err
		}
		defer listener.Close()

		containerRootfs := filepath.Join(containerPath, spec.Root.Path)

		if err := filesystem.MountProc(containerRootfs); err != nil {
			initConn.Write([]byte(fmt.Sprintf("mount proc: %s", err)))
		}

		if spec.Linux != nil && len(spec.Linux.Devices) > 0 {
			for _, dev := range spec.Linux.Devices {
				target := filepath.Join(containerRootfs, strings.TrimPrefix(dev.Path, "/"))
				fmt.Printf("mount '%s' to '%s': \n", dev.Path, target)
				// if err := syscall.Mount(
				// 	target,
				// 	target,
				// 	unix.MS_BIND,
				// 	dev.FileMode, // ???
				// ); err != nil {
				// 	// TODO: log error
				// 	initConn.Write([]byte(err.Error()))
				// }
			}

		}

		if err := filesystem.MountDefaultDevices(containerRootfs); err != nil {
			initConn.Write([]byte(fmt.Sprintf("mount dev: %s", err)))
			return err
		}

		if err := filesystem.MountRootfs(containerRootfs); err != nil {
			initConn.Write([]byte(fmt.Sprintf("mount rootfs: %s", err)))
			return err
		}

		if err := filesystem.PivotRootfs(containerRootfs); err != nil {
			initConn.Write([]byte(fmt.Sprintf("pivot root: %s", err)))
			return err
		}

		if n, err := initConn.Write([]byte("ready")); n == 0 || err != nil {
			// TODO: log error
			return err
		}

		for {
			containerConn, err := listener.Accept()
			if err != nil {
				log.Error().Err(err).Msg("accept connection")
				initConn.Write([]byte(fmt.Sprintf("accept connection: %s", err)))
				continue
			}

			go server(containerConn, spec)
		}

	default:
		return nil

	}

	return nil
}
