package commands

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type CreateOpts struct {
	ID                string
	Bundle            string
	ConsoleSocketPath string
	PIDFile           string
}

func Create(opts *CreateOpts, log *zerolog.Logger) error {
	bundle, err := internal.NewBundle(opts.Bundle)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}

	if bundle.Spec.Linux == nil {
		return errors.New("not a linux container")
	}

	container, err := internal.NewContainer(opts.ID, bundle)
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
	listener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on init socket: %w", err)
	}
	defer listener.Close()

	term := container.Spec.Process != nil && container.Spec.Process.Terminal && opts.ConsoleSocketPath != ""
	var termFD int
	if term {
		termsock, err := terminal.New(opts.ConsoleSocketPath)
		if err != nil {
			return fmt.Errorf("create terminal socket: %w", err)
		}
		termFD = termsock.FD
	}

	containerSockAddr := filepath.Join(container.Path, "container.sock")
	forkCmd := exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			opts.ID,
			initSockAddr,
			containerSockAddr,
			strconv.Itoa(container.State.Pid),
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

	forkCmd.Env = container.Spec.Process.Env

	if err := forkCmd.Start(); err != nil {
		return fmt.Errorf("fork: %w", err)
	}

	// need to get the pid off the process _before_ releasing it
	container.State.Pid = forkCmd.Process.Pid
	if err := forkCmd.Process.Release(); err != nil {
		log.Error().Err(err).Msg("detach fork")
		return err
	}

	// write pid to file if provided
	if opts.PIDFile != "" {
		pid := strconv.Itoa(container.State.Pid)
		os.WriteFile(opts.PIDFile, []byte(pid), 0666)
	}

	initConn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer initConn.Close()

	b := make([]byte, 128)

	for {
		time.Sleep(time.Second)

		n, err := initConn.Read(b)
		if err != nil || n == 0 {
			if err == io.EOF {
				fmt.Println("error: received EOF from socket")
				os.Exit(1)
			}

			fmt.Println("error: ", err)
			continue
		}

		if len(b) >= 5 && string(b[:5]) == "ready" {
			log.Info().Msg("ready")
			break
		} else {
			fmt.Println(string(b))
		}
	}

	container.State.Set(specs.StateCreated)
	if err := container.State.Save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}
