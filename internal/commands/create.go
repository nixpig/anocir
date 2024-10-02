package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nixpig/brownie/internal/bundle"
	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	ConsoleSocket string
	PIDFile       string
}

func Create(opts *CreateOpts, log *zerolog.Logger) error {
	bundle, err := bundle.New(opts.Bundle)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}

	if bundle.Spec.Linux == nil {
		return errors.New("not a linux container")
	}

	container, err := container.New(opts.ID, bundle)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := container.ExecHooks("createRuntime"); err != nil {
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := container.ExecHooks("createContainer"); err != nil {
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	container.State.Set(specs.StateCreating)
	container.State.Save()

	initSockAddr := filepath.Join(container.Path, "init.sock")
	if err := os.RemoveAll(initSockAddr); err != nil {
		return err
	}

	initCh, closer, err := ipc.NewReceiver(initSockAddr)
	if err != nil {
		return err
	}
	defer closer()

	useTerminal := container.Spec.Process != nil && container.Spec.Process.Terminal && opts.ConsoleSocket != ""
	var termFD int
	if useTerminal {
		termSock, err := terminal.New(opts.ConsoleSocket)
		if err != nil {
			return fmt.Errorf("create terminal socket: %w", err)
		}
		termFD = termSock.FD
	}

	forkCmd := exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			opts.ID,
			initSockAddr,
			strconv.Itoa(termFD),
		}...)

	// apply configuration, e.g. devices, proc, etc...
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                container.AmbientCapsFlags,
		Cloneflags:                 container.NamespaceFlags,
		Unshareflags:               syscall.CLONE_NEWNS,
		GidMappingsEnableSetgroups: false,
		UidMappings:                container.UIDMappings,
		GidMappings:                container.GIDMappings,
	}

	forkCmd.Stdin = os.Stdin
	forkCmd.Stdout = os.Stdout
	forkCmd.Stderr = os.Stderr

	forkCmd.Env = container.Spec.Process.Env

	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("fork: %w", err)
	}

	// // need to get the pid off the process _before_ releasing it
	// // FIXME: should this end up being zero??
	// container.State.Pid = forkCmd.Process.Pid
	if err := forkCmd.Process.Release(); err != nil {
		log.Error().Err(err).Msg("detach fork")
		return err
	}

	// write pid to file if provided
	// if opts.PIDFile != "" {
	// 	pid := strconv.Itoa(container.State.Pid)
	// 	os.WriteFile(opts.PIDFile, []byte(pid), 0666)
	// }

	for {
		ready := <-initCh
		if string(ready[:5]) == "ready" {
			log.Info().Msg("ready")
			break
		}
	}

	container.State.Set(specs.StateCreated)
	if err := container.State.Save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
