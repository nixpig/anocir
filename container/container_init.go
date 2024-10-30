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
	"github.com/rs/zerolog"
)

type InitOpts struct {
	PIDFile       string
	ConsoleSocket string
	Stdin         *os.File
	Stdout        *os.File
	Stderr        *os.File
}

func (c *Container) Init(opts *InitOpts, log *zerolog.Logger) error {
	initSockAddr := filepath.Join(c.State.Bundle, initSockFilename)
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
		opts.ConsoleSocket != ""

	var termFD int
	if useTerminal {
		termSock, err := terminal.New(opts.ConsoleSocket)
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

	c.forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                ambientCapsFlags,
		Cloneflags:                 cloneFlags,
		Unshareflags:               unshareFlags | syscall.CLONE_NEWNS,
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
		return fmt.Errorf("start fork container: %w", err)
	}

	pid := c.forkCmd.Process.Pid
	c.State.PID = pid
	if err := c.Save(); err != nil {
		return fmt.Errorf("save pid for fork: %w", err)
	}

	if err := c.forkCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach fork container: %w", err)
	}

	if opts.PIDFile != "" {
		log.Info().Str("pidfile", opts.PIDFile).Int("pid", pid).Msg("writing pidfile")
		if err := os.WriteFile(
			opts.PIDFile,
			[]byte(strconv.Itoa(pid)),
			0666,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", opts.PIDFile, err)
		}
	}

	return ipc.WaitForMsg(c.initIPC.ch, "ready", func() error {
		c.State.Status = specs.StateCreated
		if err := c.Save(); err != nil {
			return fmt.Errorf("save created state: %w", err)
		}
		return nil
	})

	// p, err := os.FindProcess(pid)
	// if err != nil {
	// 	log.Error().Err(err).Int("pid", pid).Msg("failed to find process")
	// 	return -1, err
	// }

	// fmt.Println("waiting for process to exit", p.Pid)
	// o, err := p.Wait()
	// if err != nil {
	// 	log.Error().Err(err).Int("pid", pid).Msg("waiting for process to exit")
	// 	return -1, err
	// }
	//
	// fmt.Println(o)

}
