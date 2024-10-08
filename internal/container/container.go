package container

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/internal/cgroups"
	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/internal/lifecycle"
	"github.com/nixpig/brownie/internal/namespace"
	"github.com/nixpig/brownie/internal/state"
	"github.com/nixpig/brownie/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	cp "github.com/otiai10/copy"
	"golang.org/x/sys/unix"
)

const configFilename = "config.json"
const initSockFilename = "init.sock"
const containerSockFilename = "container.sock"

type Container struct {
	Root  string
	State *state.State
	Spec  *specs.Spec

	forkCmd *exec.Cmd
	initIPC ipcCtrl
}

type InitOpts struct {
	PIDFile       string
	ConsoleSocket string
	Stdin         *os.File
	Stdout        *os.File
	Stderr        *os.File
}

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ConsoleSocketFD   int
	ConsoleSocketPath string
}

type ipcCtrl struct {
	ch     chan []byte
	closer func() error
}

func New(
	id string,
	bundle string,
	root string,
	status specs.ContainerState,
) (*Container, error) {
	if stat, err := os.Stat(root); stat != nil || os.IsExist(err) {
		return nil, fmt.Errorf(
			"container path already exists (%s): %w",
			root, err,
		)
	}

	b, err := os.ReadFile(filepath.Join(bundle, configFilename))
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, fmt.Errorf("parse container config: %w", err)
	}

	if err := os.MkdirAll(root, os.ModeDir); err != nil {
		return nil, fmt.Errorf("create container directory")
	}

	if err := cp.Copy(
		filepath.Join(bundle, configFilename),
		filepath.Join(root, configFilename),
	); err != nil {
		return nil, fmt.Errorf("copy config.json: %w", err)
	}

	rootfs := "rootfs"
	if spec.Root != nil {
		rootfs = spec.Root.Path
	}

	if err := cp.Copy(
		filepath.Join(bundle, rootfs),
		filepath.Join(root, rootfs),
	); err != nil {
		return nil, fmt.Errorf("copy rootfs: %w", err)
	}

	absBundlePath, err := filepath.Abs(bundle)
	if err != nil {
		return nil, fmt.Errorf("construct absolute bundle path: %w", err)
	}

	return &Container{
		Root:  root,
		State: state.New(id, absBundlePath, status),
		Spec:  &spec,
	}, nil
}

func (c *Container) Init(opts *InitOpts) (int, error) {
	initSockAddr := filepath.Join(c.Root, initSockFilename)
	if err := os.RemoveAll(initSockAddr); err != nil {
		return -1, fmt.Errorf("remove existing init socket: %w", err)
	}

	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewReceiver(initSockAddr)
	if err != nil {
		return -1, fmt.Errorf("create init ipc receiver: %w", err)
	}
	defer c.initIPC.closer()

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		opts.ConsoleSocket != ""

	var termFD int
	if useTerminal {
		termSock, err := terminal.New(opts.ConsoleSocket)
		if err != nil {
			return -1, fmt.Errorf("create terminal socket: %w", err)
		}
		termFD = termSock.FD
	}

	c.forkCmd = exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			c.State.ID,
			initSockAddr,
			strconv.Itoa(termFD),
		}...)

	var ambientCapsFlags []uintptr
	if c.Spec.Process != nil &&
		c.Spec.Process.Capabilities != nil {
		for _, cap := range c.Spec.Process.Capabilities.Ambient {
			ambientCapsFlags = append(
				ambientCapsFlags,
				uintptr(capabilities.Capabilities[cap]),
			)
		}
	}

	var cloneFlags uintptr
	if c.Spec.Linux != nil && c.Spec.Linux.Namespaces != nil {
		for _, ns := range c.Spec.Linux.Namespaces {
			ns := namespace.LinuxNamespace(ns)
			flag, err := ns.ToFlag()
			if err != nil {
				return -1, fmt.Errorf("convert namespace to flag: %w", err)
			}

			cloneFlags |= flag
		}
	}

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap
	if c.Spec.Process != nil {
		cloneFlags |= syscall.CLONE_NEWUSER

		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: int(c.Spec.Process.User.UID),
			HostID:      os.Geteuid(),
			Size:        1,
		})

		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: int(c.Spec.Process.User.GID),
			HostID:      os.Getegid(),
			Size:        1,
		})
	}

	if c.Spec.Linux.UIDMappings != nil {
		for _, uidMapping := range c.Spec.Linux.UIDMappings {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: int(uidMapping.ContainerID),
				HostID:      int(uidMapping.HostID),
				Size:        int(uidMapping.Size),
			})
		}
	}

	if c.Spec.Linux.GIDMappings != nil {
		for _, gidMapping := range c.Spec.Linux.GIDMappings {
			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: int(gidMapping.ContainerID),
				HostID:      int(gidMapping.HostID),
				Size:        int(gidMapping.Size),
			})
		}
	}

	c.forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                ambientCapsFlags,
		Cloneflags:                 cloneFlags,
		Unshareflags:               syscall.CLONE_NEWNS,
		GidMappingsEnableSetgroups: false,
		UidMappings:                uidMappings,
		GidMappings:                gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		c.forkCmd.Env = c.Spec.Process.Env
	}

	c.forkCmd.Stdin = opts.Stdin
	c.forkCmd.Stdout = opts.Stdout
	c.forkCmd.Stderr = opts.Stderr

	if err := c.forkCmd.Start(); err != nil {
		return -1, fmt.Errorf("start fork container: %w", err)
	}

	pid := c.forkCmd.Process.Pid
	c.State.PID = pid
	if err := c.State.Save(c.Root); err != nil {
		return -1, fmt.Errorf("save state for fork: %w", err)
	}

	if err := c.forkCmd.Process.Release(); err != nil {
		return -1, fmt.Errorf("detach fork container: %w", err)
	}

	if opts.PIDFile != "" {
		if err := os.WriteFile(
			opts.PIDFile,
			[]byte(strconv.Itoa(pid)),
			0666,
		); err != nil {
			return -1, fmt.Errorf("write pid to file (%s): %w", opts.PIDFile, err)
		}
	}

	for {
		ready := <-c.initIPC.ch

		if string(ready[:5]) == "ready" {
			break
		}
	}

	c.State.Status = specs.StateCreated
	if err := c.State.Save(c.Root); err != nil {
		return -1, fmt.Errorf("save created state: %w", err)
	}

	return pid, nil
}

func (c *Container) Fork(opts *ForkOpts) error {
	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		return err
	}
	defer c.initIPC.closer()

	if opts.ConsoleSocketFD != 0 {
		pty, err := terminal.NewPty()
		if err != nil {
			return err
		}
		defer pty.Close()

		if err := pty.Connect(); err != nil {
			return err
		}

		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// FIXME: how do we pass ptysocket struct between fork?
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			return err
		}
	}

	rootfs := "rootfs"
	if c.Spec.Root != nil {
		rootfs = c.Spec.Root.Path
	}

	rootfs = filepath.Join(c.Root, rootfs)

	if err := filesystem.MountRootfs(rootfs); err != nil {
		return err
	}

	if err := filesystem.MountProc(rootfs); err != nil {
		return err
	}

	if err := filesystem.MountSpecMounts(
		c.Spec.Mounts,
		rootfs,
	); err != nil {
		return err
	}

	if err := filesystem.MountDevices(
		filesystem.DefaultDevices,
		rootfs,
	); err != nil {
		return err
	}

	for _, dev := range c.Spec.Linux.Devices {
		var absPath string
		if strings.Index(dev.Path, "/") == 0 {
			relPath := strings.TrimPrefix(dev.Path, "/")
			absPath = filepath.Join(rootfs, relPath)
		} else {
			absPath = filepath.Join(rootfs, dev.Path)
		}

		if err := unix.Mknod(
			absPath,
			uint32(*dev.FileMode),
			int(unix.Mkdev(uint32(dev.Major), uint32(dev.Minor))),
		); err != nil {
			return err
		}

		if err := os.Chown(
			absPath,
			int(*dev.UID),
			int(*dev.GID),
		); err != nil {
			return err
		}
	}

	if err := filesystem.CreateSymlinks(
		filesystem.DefaultSymlinks,
		rootfs,
	); err != nil {
		return err
	}

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join(c.Root, containerSockFilename),
	); err != nil {
		return err
	}

	listener, err := net.Listen(
		"unix",
		filepath.Join(c.Root, containerSockFilename),
	)
	if err != nil {
		return err
	}
	defer listener.Close()

	if err := filesystem.PivotRootfs(rootfs); err != nil {
		return err
	}

	if slices.ContainsFunc(
		c.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname(
			[]byte(c.Spec.Hostname),
		); err != nil {
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(c.Spec.Domainname),
		); err != nil {
			return err
		}
	}

	if err := capabilities.SetCapabilities(
		c.Spec.Process.Capabilities,
	); err != nil {
		return err
	}

	if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
		return err
	}

	c.initIPC.ch <- []byte("ready")

	containerConn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer containerConn.Close()

	for {
		b := make([]byte, 128)

		n, _ := containerConn.Read(b)
		// if err != nil {
		// 	log.Error().Err(err).Msg("read from connection")
		// 	return err
		// }
		//
		if n == 0 {
			continue
		}

		switch string(b[:n]) {
		case "start":
			cmd := exec.Command(
				c.Spec.Process.Args[0],
				c.Spec.Process.Args[1:]...,
			)

			cmd.Dir = c.Spec.Process.Cwd

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return err
			}
		}
	}
}

func Load(root string) (*Container, error) {
	state, err := state.Load(root)
	if err != nil {
		return nil, fmt.Errorf("load container state: %w", err)
	}

	b, err := os.ReadFile(filepath.Join(root, configFilename))
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, fmt.Errorf("parse container config: %w", err)
	}

	return &Container{
		Root:  root,
		State: state,
		Spec:  &spec,
	}, nil
}

func (c *Container) Save() error {
	return c.State.Save(c.Root)
}

func (c *Container) Clean() error {
	return os.RemoveAll(c.Root)
}

func (c *Container) ExecHooks(hook string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var specHooks []specs.Hook
	switch hook {
	case "createRuntime":
		specHooks = c.Spec.Hooks.CreateRuntime
	case "createContainer":
		specHooks = c.Spec.Hooks.CreateContainer
	case "startContainer":
		specHooks = c.Spec.Hooks.StartContainer
	case "poststart":
		specHooks = c.Spec.Hooks.Poststart
	case "poststop":
		specHooks = c.Spec.Hooks.Poststop
	}

	return lifecycle.ExecHooks(specHooks)
}

func (c *Container) CanBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func (c *Container) CanBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateStopped
}

func (c *Container) CanBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}
