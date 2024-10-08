package commands

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/nixpig/brownie/internal/container"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger) error {
	root := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)

	cntr, err := container.Load(root)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	if !cntr.CanBeStarted() {
		log.Error().Msg("container cannot be started in current state")
		return errors.New("container cannot be started in current state")
	}

	if err := cntr.ExecHooks("startContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute startContainer hooks")
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", filepath.Join(root, containerSockFilename))
	if err != nil {
		log.Error().Err(err).Msg("failed to dial container sockaddr")
		return fmt.Errorf("dial socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("failed to send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	// FIXME: ?? when process starts, the PID in state should be updated to the process IN the container??

	cntr.State.Status = specs.StateRunning
	if err := cntr.State.Save(root); err != nil {
		log.Error().Err(err).Msg("failed to save state")
	}

	if err := cntr.ExecHooks("poststart"); err != nil {
		log.Warn().Err(err).Msg("failed to execute poststart hooks")
	}

	cntr.State.Status = specs.StateStopped
	if err := cntr.State.Save(root); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
