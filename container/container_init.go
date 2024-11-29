package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/nixpig/brownie/container/cgroups"
	"github.com/nixpig/brownie/container/namespace"
	"github.com/nixpig/brownie/container/terminal"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Init(reexec string, arg string, log *zerolog.Logger) error {
	if err := c.ExecHooks("createRuntime", log); err != nil {
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := c.ExecHooks("createContainer", log); err != nil {
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	initSockAddr := filepath.Join("/var/lib/brownie/containers", c.ID(), initSockFilename)

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

	// -------------------------------
	// TODO: apply cgroups
	// -------------------------------

	if c.Spec.Linux.CgroupsPath != "" && c.Spec.Linux.Resources != nil {
		staticPath := cgroup1.StaticPath(c.Spec.Linux.CgroupsPath)

		cg, err := cgroup1.New(
			staticPath,
			&specs.LinuxResources{
				Devices: c.Spec.Linux.Resources.Devices,
			},
		)
		if err != nil {
			return fmt.Errorf("apply cgroups (path: %s): %w", c.Spec.Linux.CgroupsPath, err)
		}
		defer cg.Delete()

		cg.Add(cgroup1.Process{Pid: c.PID()})
	}

	// -------------------------------
	// -------------------------------
	// -------------------------------

	reexecCmd := exec.Command(
		reexec,
		[]string{arg, "--stage", "1", c.ID()}...,
	)

	cloneFlags := uintptr(0)
	for _, ns := range c.Spec.Linux.Namespaces {
		if ns.Type == "user" {
			continue
		}
		ns := namespace.LinuxNamespace(ns)
		flag, err := ns.ToFlag()
		if err != nil {
			return fmt.Errorf("convert namespace to flag: %w", err)
		}

		// if it's path-based, we need to do it in the reexec
		if ns.Path == "" {
			cloneFlags |= flag
		}
	}

	reexecCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   cloneFlags,
		Unshareflags: uintptr(0),
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		reexecCmd.Env = c.Spec.Process.Env
	}

	if c.Spec.Process != nil && c.Spec.Process.Rlimits != nil {
		for _, rl := range c.Spec.Process.Rlimits {
			if _, ok := cgroups.Rlimits[rl.Type]; !ok {
				return errors.New("unable to map rlimit to kernel interface")
			}
		}
	}

	reexecCmd.Stdin = c.Opts.Stdin
	reexecCmd.Stdout = c.Opts.Stdout
	reexecCmd.Stderr = c.Opts.Stderr

	if err := reexecCmd.Start(); err != nil {
		return fmt.Errorf("start reexec container: %w", err)
	}

	pid := reexecCmd.Process.Pid
	c.SetPID(pid)
	if err := c.Save(); err != nil {
		return fmt.Errorf("save pid for reexec: %w", err)
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

	if err := reexecCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach reexec container: %w", err)
	}

	return ipc.WaitForMsg(c.initIPC.ch, "ready", func() error {
		c.SetStatus(specs.StateCreated)
		if err := c.Save(); err != nil {
			return fmt.Errorf("save created state: %w", err)
		}
		return nil
	})
}
