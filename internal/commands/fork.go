package commands

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
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
	specJSON, err := os.ReadFile(filepath.Join(containerPath, "config.json"))
	if err != nil {
		log.Error().Err(err).Msg("read spec")
		return err
	}

	var spec specs.Spec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		log.Error().Err(err).Msg("parse config")
		return err
	}

	switch opts.Stage {
	case ForkIntermediate:
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

		containerSockAddr := filepath.Join(containerPath, "container.sock")

		forkCmd := exec.Command(
			"/proc/self/exe",
			[]string{
				"fork",
				string(ForkDetached),
				opts.ID,
				opts.InitSockAddr,
				containerSockAddr,
			}...)

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

		log.Info().Msg("start second fork - detached")
		if err := forkCmd.Start(); err != nil {
			log.Error().Err(err).Msg("detached fork start")
			return err
		}

		// pid := forkCmd.Process.Pid
		log.Info().Msg("release second fork process")
		if err := forkCmd.Process.Release(); err != nil {
			log.Error().Err(err).Msg("detach fork")
			return err
		}
		log.Info().Msg("process successfully released")

	case ForkDetached:
		initConn, err := net.Dial("unix", opts.InitSockAddr)
		if err != nil {
			log.Error().Err(err).Msg("dialing init socket")
			return err
		}
		defer initConn.Close()
		log.Info().Msg("IN THE DETACHED FORK!!")
		if err := os.RemoveAll(opts.ContainerSockAddr); err != nil {
			log.Error().Err(err).Msg("remove existing container socket")
			return err
		}

		log.Info().Msg("listen on container socket")
		listener, err := net.Listen("unix", opts.ContainerSockAddr)
		if err != nil {
			log.Error().Err(err).Msg("listen on container socket")
			return err
		}
		defer listener.Close()

		containerRootfs := filepath.Join(containerPath, spec.Root.Path)

		// if err := filesystem.MountProc(containerRootfs); err != nil {
		// 	initConn.Write([]byte(fmt.Sprintf("mount proc: %s", err)))
		// }

		log.Info().Msg("mount rootfs")
		if err := filesystem.MountRootfs(containerRootfs); err != nil {
			log.Error().Err(err).Msg("mount rootfs")
			return err
		}

		log.Info().Msg("mount spec mounts")
		for _, mount := range spec.Mounts {
			var dest string
			if strings.Index(mount.Destination, "/") == 0 {
				dest = containerRootfs + mount.Destination
			} else {
				dest = mount.Destination
			}

			log.Info().Str("dest", dest).Msg("check if spec mount destination exists")
			if _, err := os.Stat(dest); err != nil {

				log.Error().Err(err).Msg("check if spec mount destination exists")
				if !os.IsNotExist(err) {
					return err
				}

				log.Info().Str("dest", dest).Msg("create directory for spec mount")
				if err := os.MkdirAll(dest, os.ModeDir); err != nil {
					log.Error().Err(err).Str("dest", dest).Msg("make spec mount dir")
					return err
				}
			}

			var flags uintptr
			if mount.Type == "bind" {
				flags = flags | syscall.MS_BIND
			}

			var data string

			if len(mount.Options) > 0 {
				data = strings.Join(mount.Options, ",")
			}

			if err := syscall.Mount(
				mount.Source,
				dest,
				mount.Type,
				flags,
				"",
			); err != nil {
				c, _ := user.Current()
				log.Error().Err(err).
					Str("source", mount.Source).
					Str("dest", dest).
					Str("options", data).
					Any("user", c).
					Msg("mount spec mount")
				return err
			}
		}

		log.Info().Msg("create spec devices")
		for _, dev := range spec.Linux.Devices {
			var absPath string
			if strings.Index(dev.Path, "/") == 0 {
				relPath := strings.TrimPrefix(dev.Path, "/")
				absPath = filepath.Join(containerRootfs, relPath)
			} else {
				absPath = filepath.Join(containerRootfs, dev.Path)
			}

			if err := unix.Mknod(
				absPath,
				uint32(*dev.FileMode),
				int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor))),
			); err != nil {
				log.Error().Err(err).Any("dev", dev).Msg("make device node")
				return err
			}

			log.Info().Msg("chown device node")
			if err := os.Chown(absPath, int(*dev.UID), int(*dev.GID)); err != nil {
				log.Error().Err(err).Str("path", absPath).Uint32("UID", *dev.UID).Uint32("GID", *dev.GID).Msg("chown device node")
				return err
			}
		}

		log.Info().Msg("create default devices")
		for _, dev := range filesystem.DefaultDevices {
			var absPath string
			if strings.Index(dev.Path, "/") == 0 {
				relPath := strings.TrimPrefix(dev.Path, "/")
				absPath = filepath.Join(containerRootfs, relPath)
			} else {
				absPath = filepath.Join(containerRootfs, dev.Path)
			}

			if err := unix.Mknod(
				absPath,
				uint32(*dev.FileMode),
				int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor))),
			); err != nil {
				log.Error().Err(err).Msg("mount default device")
				return err
			}

			if err := os.Chown(absPath, int(*dev.UID), int(*dev.GID)); err != nil {
				log.Error().Err(err).Str("path", absPath).Uint32("UID", *dev.UID).Uint32("GID", *dev.GID).Msg("chown default device")
				return err
			}
		}

		log.Info().Msg("create default symlinks")
		for oldname, newname := range filesystem.DefaultSymlinks {
			if err := os.Symlink(oldname, filepath.Join(containerRootfs, newname)); err != nil {
				log.Error().Err(err).Str("newname", newname).Str("oldname", oldname).Msg("create default symlink")
				return err
			}
		}

		log.Info().Msg("set hostname")
		if err := syscall.Sethostname([]byte(spec.Hostname)); err != nil {
			log.Error().Err(err).Str("hostname", spec.Hostname).Msg("set hostname")
			return err
		}

		log.Info().Msg("pivot rootfs")
		if err := filesystem.PivotRootfs(containerRootfs); err != nil {
			log.Error().Err(err).Msg("pivot rootfs")
			return err
		}

		time.Sleep(time.Second * 1) // give the listener in 'create' time to come up
		log.Info().Str("socket", initConn.RemoteAddr().String()).Msg("send 'ready' message")
		if n, err := initConn.Write([]byte("ready")); n == 0 || err != nil {
			log.Error().Err(err).Msg("send 'ready' message")
			return err
		}

		log.Info().Msg("wait for message...")
		for {
			containerConn, err := listener.Accept()
			if err != nil {
				log.Error().Err(err).Msg("accept connection")
				continue
			}

			go server(containerConn, spec)
		}

	default:
		return nil

	}

	return nil
}
