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
	log.Info().Any("opts", opts).Msg("run create command")
	log.Info().Str("bundle", opts.Bundle).Msg("create bundle")
	bundle, err := bundle.New(opts.Bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to create bundle")
		return fmt.Errorf("create bundle: %w", err)
	}

	if bundle.Spec.Linux == nil {
		log.Error().Msg("not a linux container")
		return errors.New("not a linux container")
	}

	log.Info().
		Str("id", opts.ID).
		Str("bundle", opts.Bundle).
		Msg("create container from bundle")
	cntr, err := container.New(opts.ID, bundle)
	if err != nil {
		log.Error().Err(err).Msg("failed to create container from bundle")
		return fmt.Errorf("create container: %w", err)
	}

	log.Info().Msg("execute createRuntime hooks")
	if err := cntr.ExecHooks("createRuntime"); err != nil {
		log.Error().Err(err).Msg("failed to execute createRuntime hooks")
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	log.Info().Msg("execute createContainer hooks")
	if err := cntr.ExecHooks("createContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute createContainer hooks")
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	log.Info().
		Any("state", cntr.State.Status).
		Msg("set and save creating state")
	cntr.State.Set(specs.StateCreating)
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save creating state")
		return fmt.Errorf("save creating state: %w", err)
	}

	initSockAddr := filepath.Join(cntr.Path, "init.sock")
	log.Info().Str("sockaddr", initSockAddr).Msg("remove existing init sockaddr")
	if err := os.RemoveAll(initSockAddr); err != nil {
		log.Error().Err(err).Msg("failed to remove existing sockaddr")
		return fmt.Errorf("remove existing init socket: %w", err)
	}

	log.Info().Str("sockaddr", initSockAddr).Msg("create new ipc receiver")
	initCh, initCloser, err := ipc.NewReceiver(initSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed to create init ipc receiver")
		return fmt.Errorf("create init ipc receiver: %w", err)
	}
	defer initCloser()

	useTerminal := cntr.Spec.Process != nil &&
		cntr.Spec.Process.Terminal &&
		opts.ConsoleSocket != ""

	var termFD int
	if useTerminal {
		log.Info().Str("console", opts.ConsoleSocket).Msg("create new terminal")
		termSock, err := terminal.New(opts.ConsoleSocket)
		if err != nil {
			log.Error().Err(err).Msg("failed to create new terminal")
			return fmt.Errorf("create terminal socket: %w", err)
		}
		termFD = termSock.FD
	}

	log.Info().Str("name", "/proc/self/exe").Msg("create fork exec command")
	forkCmd := exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			opts.ID,
			initSockAddr,
			strconv.Itoa(termFD),
		}...)

	log.Info().Msg("set sysprocattr on fork exec command")
	forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                cntr.AmbientCapsFlags,
		Cloneflags:                 cntr.NamespaceFlags,
		Unshareflags:               syscall.CLONE_NEWNS,
		GidMappingsEnableSetgroups: false,
		UidMappings:                cntr.UIDMappings,
		GidMappings:                cntr.GIDMappings,
	}

	log.Info().
		Str("stdin", os.Stdin.Name()).
		Str("stdout", os.Stdout.Name()).
		Str("stderr", os.Stderr.Name()).
		Msg("set stdio on fork exec command")
	forkCmd.Stdin = os.Stdin
	forkCmd.Stdout = os.Stdout
	forkCmd.Stderr = os.Stderr

	log.Info().Any("env", cntr.Spec.Process.Env).Msg("set spec environment")
	forkCmd.Env = cntr.Spec.Process.Env

	log.Info().Msg("start fork exec")
	if err := forkCmd.Start(); err != nil {
		log.Error().Err(err).Msg("failed to start fork exec")
		return fmt.Errorf("start fork: %w", err)
	}

	log.Info().Msg("release fork exec process")
	if err := forkCmd.Process.Release(); err != nil {
		log.Error().Err(err).Msg("failed to release fork exec process")
		return err
	}

	if opts.PIDFile != "" {
		log.Info().Msg("write pid to file")
		pid := strconv.Itoa(cntr.State.Pid)
		if err := os.WriteFile(opts.PIDFile, []byte(pid), 0666); err != nil {
			log.Error().Err(err).Msg("failed to write pid to file")
			return fmt.Errorf("write pid to file: %w", err)
		}
	}

	log.Info().Msg("waiting for ready message on init")
	for {
		ready := <-initCh
		log.Info().
			Str("msg", string(ready)).
			Msg("host received ipc msg")

		if string(ready[:5]) == "ready" {
			log.Info().Msg("received ready message")
			break
		}
	}

	log.Info().
		Any("state", cntr.State.Status).
		Msg("set and save created state")
	cntr.State.Set(specs.StateCreated)
	if err := cntr.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save created state")
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
