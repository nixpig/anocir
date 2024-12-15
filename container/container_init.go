package container

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/cgroups"
	"github.com/nixpig/brownie/namespace"
	"github.com/nixpig/brownie/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Init(reexec string, arg string) error {
	if err := c.ExecHooks("createRuntime"); err != nil {
		return fmt.Errorf("execute createRuntime hooks: %w", err)
	}

	if err := c.ExecHooks("createContainer"); err != nil {
		return fmt.Errorf("execute createContainer hooks: %w", err)
	}

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		c.Opts.ConsoleSocket != ""

	if useTerminal {
		var err error
		if c.State.ConsoleSocket, err = terminal.Setup(
			c.Rootfs(),
			c.Opts.ConsoleSocket,
		); err != nil {
			return err
		}
	}

	if c.Spec.Linux.CgroupsPath != "" && c.Spec.Linux.Resources != nil {
		if err := cgroups.AddV1(
			c.Spec.Linux.CgroupsPath,
			c.Spec.Linux.Resources.Devices,
			c.PID(),
		); err != nil {
			return err
		}
	}

	// ---------------------------

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := os.WriteFile(
			"/proc/self/oom_score_adj",
			[]byte(strconv.Itoa(*c.Spec.Process.OOMScoreAdj)),
			0644,
		); err != nil {
			return fmt.Errorf("create oom score adj file: %w", err)
		}
	}

	if c.State.ConsoleSocket != nil {
		pty, err := terminal.NewPty()
		if err != nil {
			return fmt.Errorf("new pty: %w", err)
		}

		if err := pty.Connect(); err != nil {
			return fmt.Errorf("connect pty: %w", err)
		}

		if err := terminal.SendPty(
			*c.State.ConsoleSocket,
			pty,
		); err != nil {
			return fmt.Errorf("connect pty and socket: %w", err)
		}

	} else {
		// TODO: fall back to dup2 on stdin, stdout, stderr from c.Opts??
		fmt.Println("TODO: implement fallback stdio??")
	}

	// ---------------------------

	reexecCmd := exec.Command(
		reexec,
		[]string{arg, c.ID()}...,
	)

	cloneFlags := uintptr(0)

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap

	for _, ns := range c.Spec.Linux.Namespaces {
		if ns.Type == specs.UserNamespace {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			})

			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			})
		}

		ns := namespace.LinuxNamespace(ns)

		if ns.Path == "" {
			cloneFlags |= ns.ToFlag()
		} else {
			if !strings.HasSuffix(ns.Path, fmt.Sprintf("/%s", ns.ToEnv())) &&
				ns.Type != specs.PIDNamespace {
				return fmt.Errorf("namespace type (%s) and path (%s) do not match", ns.Type, ns.Path)
			}

			// TODO: align so the same mechanism is used for all namespaces?
			if ns.Type == specs.MountNamespace {
				reexecCmd.Env = append(reexecCmd.Env, fmt.Sprintf("gons_%s=%s", ns.ToEnv(), ns.Path))
			} else {
				if err := ns.Enter(); err != nil {
					return fmt.Errorf("enter namespace: %w", err)
				}
			}
		}
	}

	reexecCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   cloneFlags,
		Unshareflags: uintptr(0),
		UidMappings:  uidMappings,
		GidMappings:  gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		reexecCmd.Env = append(reexecCmd.Env, c.Spec.Process.Env...)
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

	if err := reexecCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach reexec container: %w", err)
	}

	// wait for "ready"
	initSockAddr := filepath.Join(containerRootDir, c.ID(), initSockFilename)

	listener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("create init sock receiver: %w", err)
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept on listener: %w", err)
	}
	defer conn.Close()

	b := make([]byte, 128)
	n, err := conn.Read(b)
	if err != nil {
		return fmt.Errorf("read from init socket: %w", err)
	}

	msg := string(b[:n])
	if msg != "ready" {
		return fmt.Errorf("expecting 'ready', received '%s'", msg)
	}

	// after receiving "ready"
	c.SetStatus(specs.StateCreated)
	if err := c.Save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
