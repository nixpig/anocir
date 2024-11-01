package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nixpig/brownie/container/capabilities"
	"github.com/nixpig/brownie/container/namespace"
	"github.com/nixpig/brownie/container/terminal"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Init(reexec string, arg string) error {
	if err := c.ExecHooks("createRuntime"); err != nil {
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := c.ExecHooks("createContainer"); err != nil {
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	initSockAddr := filepath.Join(c.Bundle(), initSockFilename)
	if err := os.RemoveAll(initSockAddr); err != nil {
		return fmt.Errorf("remove existing init socket: %w", err)
	}

	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewReceiver(initSockAddr)
	if err != nil {
		return fmt.Errorf("create init ipc receiver: %w", err)
	}
	defer c.initIPC.closer()

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		c.Opts.ConsoleSocket != ""

	if useTerminal {
		termSock, err := terminal.New(c.Opts.ConsoleSocket)
		if err != nil {
			return fmt.Errorf("create terminal socket: %w", err)
		}
		c.termFD = &termSock.FD
	}

	forkCmd := exec.Command(
		reexec,
		[]string{
			arg,
			c.ID(),
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
	if c.Spec.Linux.Namespaces != nil {
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

	var unshareFlags uintptr

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

	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                ambientCapsFlags,
		Cloneflags:                 cloneFlags,
		Unshareflags:               unshareFlags | syscall.CLONE_NEWNS,
		GidMappingsEnableSetgroups: false,
		UidMappings:                uidMappings,
		GidMappings:                gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		forkCmd.Env = c.Spec.Process.Env
	}

	forkCmd.Stdin = c.Opts.Stdin
	forkCmd.Stdout = c.Opts.Stdout
	forkCmd.Stderr = c.Opts.Stderr

	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("start fork container: %w", err)
	}

	pid := forkCmd.Process.Pid
	c.SetPID(pid)
	if err := c.HSave(); err != nil {
		return fmt.Errorf("save pid for fork: %w", err)
	}

	if err := forkCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach fork container: %w", err)
	}

	if c.Opts.PIDFile != "" {
		if err := os.WriteFile(
			c.Opts.PIDFile,
			[]byte(strconv.Itoa(pid)),
			0666,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", c.Opts.PIDFile, err)
		}
	}

	return ipc.WaitForMsg(c.initIPC.ch, "ready", func() error {
		c.SetStatus(specs.StateCreated)
		if err := c.HSave(); err != nil {
			return fmt.Errorf("save created state: %w", err)
		}
		return nil
	})
}
