package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/internal/lifecycle"
	"github.com/nixpig/brownie/internal/namespace"
	"github.com/nixpig/brownie/internal/state"
	"github.com/nixpig/brownie/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	cp "github.com/otiai10/copy"
)

const configFilename = "config.json"
const initSockFilename = "init.sock"

type Container struct {
	Root  string
	State *state.State
	Spec  *specs.Spec

	forkCmd *exec.Cmd
	initIPC ipcCtrl
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

	if err := os.MkdirAll(root, os.ModeDir); err != nil {
		return nil, fmt.Errorf("create container directory")
	}

	if err := cp.Copy(
		filepath.Join(bundle, configFilename),
		filepath.Join(root, configFilename),
	); err != nil {
		return nil, fmt.Errorf("copy config.json: %w", err)
	}

	b, err := os.ReadFile(filepath.Join(root, configFilename))
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, fmt.Errorf("parse container config: %w", err)
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

	return &Container{
		Root:  root,
		State: state.New(id, bundle, status),
		Spec:  &spec,
	}, nil
}

func (c *Container) Init(
	consoleSocket string,
) error {
	initSockAddr := filepath.Join(c.Root, initSockFilename)
	if err := os.RemoveAll(initSockAddr); err != nil {
		return fmt.Errorf("remove existing init socket: %w", err)
	}

	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewReceiver(initSockAddr)
	if err != nil {
		return fmt.Errorf("create init ipc receiver: %w", err)
	}

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		consoleSocket != ""

	var termFD int
	if useTerminal {
		termSock, err := terminal.New(consoleSocket)
		if err != nil {
			return fmt.Errorf("create terminal socket: %w", err)
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
				return fmt.Errorf("convert namespace to flag: %w", err)
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

	return nil
}

func (c *Container) Fork(
	pidFile string,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File,
) (int, error) {
	c.forkCmd.Stdin = stdin
	c.forkCmd.Stdout = stdout
	c.forkCmd.Stderr = stderr

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

	if pidFile != "" {
		if err := os.WriteFile(
			pidFile,
			[]byte(strconv.Itoa(pid)),
			0666,
		); err != nil {
			return -1, fmt.Errorf("write pid to file (%s): %w", pidFile, err)
		}
	}

	for {
		ready := <-c.initIPC.ch

		if string(ready[:5]) == "ready" {
			break
		}
	}

	if err := c.initIPC.closer(); err != nil {
		return -1, fmt.Errorf("close init ipc: %w", err)
	}

	c.State.Status = specs.StateCreated
	if err := c.State.Save(c.Root); err != nil {
		return -1, fmt.Errorf("save created state: %w", err)
	}

	return pid, nil
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
