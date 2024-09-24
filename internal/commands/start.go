package commands

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/nixpig/brownie/internal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(
	ch chan []byte,
	opts *StartOpts,
	log *zerolog.Logger,
	stdout io.Writer,
	stderr io.Writer,
) error {
	container, err := internal.LoadContainer(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !container.CanBeStarted() {
		return errors.New("container not created")
	}

	if err := container.ExecHooks("startContainer"); err != nil {
		return fmt.Errorf("execute startcontainer hooks: %w", err)
	}

	// 8. TODO: Run the user-specified program from 'process' in the container
	// and update state to Running
	conn, err := net.Dial("unix", container.SockAddr)
	if err != nil {
		log.Error().Err(err).Msg("start: dial socket")
		return fmt.Errorf("dial socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("start: send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}

	container.State.Set(specs.StateRunning)
	if err := container.State.Save(); err != nil {
		log.Error().Err(err).Msg("failed to save state")
	}

	b, err := io.ReadAll(conn)
	if err != nil {
		log.Error().Err(err).Msg("start: read response")
		stderr.Write([]byte(err.Error()))
		return fmt.Errorf("reading response from socket: %w", err)
	}

	// FIXME: how do we redirect this to the stdout of the calling process?
	// E.g. when being run in tests.
	if _, err := stdout.Write(b); err != nil {
		log.Error().Err(err).Msg("write to stdout")
	}

	go func() {
		ch <- b
	}()

	if err := container.ExecHooks("poststart"); err != nil {
		return fmt.Errorf("execute poststart hooks: %w", err)
	}

	container.State.Set(specs.StateStopped)
	if err := container.State.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
