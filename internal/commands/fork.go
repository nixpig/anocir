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

	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

func server(conn net.Conn, containerID string, spec specs.Spec, log *zerolog.Logger) {
	defer conn.Close()

	b := make([]byte, 128)

	for {
		log.Info().Msg("reading in fork...")
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
			log.Info().Msg("fork received 'start' command")
			log.Info().Str("command", spec.Process.Args[0]).Msg("command")
			log.Info().Str("args", strings.Join(spec.Process.Args[:1], ",")).Msg("args")
			cmd := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)

			log.Info().Msg("opening out.txt for writing")
			f, err := os.OpenFile(
				"out.txt",
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0644,
			)
			if err != nil {
				log.Error().Err(err).Msg("open out file")
			}
			abs, _ := filepath.Abs(f.Name())
			log.Info().Str("name", abs).Msg("out file")
			cmd.Stdout = conn
			cmd.Stderr = conn
			if err := cmd.Run(); err != nil {
				log.Error().Err(err).Msg("error executing underlying command")
				conn.Write([]byte(err.Error()))
			} else {
				conn.Write([]byte("apparently executed successfully???"))
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
}

func Fork(opts *ForkOpts, log *zerolog.Logger) error {
	log.Info().Msg("IN THE DETACHED FORK!!")
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

	initConn, err := net.Dial("unix", opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("dialing init socket")
		return err
	}
	defer initConn.Close()
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
			if !os.IsNotExist(err) {
				log.Error().Err(err).Msg("check if spec mount destination exists")
				return err
			}

			log.Info().Str("dest", dest).Msg("spec mount dest doesn't exist")
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

		var dataOptions []string
		for _, opt := range mount.Options {
			o, ok := pkg.MountOptions[opt]
			if !ok {
				if !strings.HasPrefix(opt, "gid=") &&
					!strings.HasPrefix(opt, "uid=") &&
					opt != "newinstance" {
					dataOptions = append(dataOptions, opt)
				}
			} else {
				if !o.No {
					flags = flags | o.Flag
				}
			}
		}

		var data string
		if len(dataOptions) > 0 {
			data = strings.Join(dataOptions, ",")
		}

		if err := syscall.Mount(
			mount.Source,
			dest,
			mount.Type,
			flags,
			data,
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

	log.Info().Msg("link default file descriptors")
	for oldname, newname := range filesystem.DefaultFileDescriptors {
		nn := filepath.Join(containerRootfs, newname)
		log.Info().Str("newname", nn).Str("oldname", oldname).Msg("link default file descriptor")
		if err := os.Symlink(oldname, nn); err != nil {
			log.Error().Err(err).Str("newname", newname).Str("oldname", oldname).Msg("link default file descriptors")
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

		go server(containerConn, opts.ID, spec, log)
	}

}
