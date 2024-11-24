package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nixpig/brownie/container/cgroups"
	"github.com/nixpig/brownie/container/namespace"
	"github.com/nixpig/brownie/container/terminal"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Init(reexec string, arg string, log *zerolog.Logger) error {
	log.Info().Msg("init")
	if err := c.ExecHooks("createRuntime"); err != nil {
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := c.ExecHooks("createContainer"); err != nil {
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	initSockAddr := filepath.Join(c.Bundle(), initSockFilename)

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

	log.Info().Msg("create command")
	reexecCmd := exec.Command(
		reexec,
		[]string{arg, "--stage", "1", c.ID()}...,
	)

	cloneFlags := uintptr(0)
	for _, ns := range c.Spec.Linux.Namespaces {
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

	log.Info().Msg("start")
	if err := reexecCmd.Start(); err != nil {
		return fmt.Errorf("start reexec container: %w", err)
	}

	pid := reexecCmd.Process.Pid
	c.SetPID(pid)
	if err := c.HSave(); err != nil {
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

	log.Info().Msg("release")
	if err := reexecCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach reexec container: %w", err)
	}

	return ipc.WaitForMsg(c.initIPC.ch, "ready", func() error {
		log.Info().Msg("ready!!")
		c.SetStatus(specs.StateCreated)
		if err := c.HSave(); err != nil {
			return fmt.Errorf("save created state: %w", err)
		}
		return nil
	})
}
