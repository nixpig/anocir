package commands

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/nixpig/brownie/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

const containerSockFilename = "container.sock"

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger, db *sql.DB) error {
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		log.Error().Err(err).Str("id", opts.ID).Msg("failed to load container")
		return fmt.Errorf("load container: %w", err)
	}

	if cntr.Spec.Process == nil {
		log.Info().Msg("NO PROCESS SET!!")
		cntr.State.Status = specs.StateStopped
		if err := cntr.Save(); err != nil {
			log.Error().Err(err).Msg("failed to write state file")
			return fmt.Errorf("write state file: %w", err)
		}
		return nil
	}

	if !cntr.CanBeStarted() {
		log.Error().Msg("container cannot be started in current state")
		return errors.New("container cannot be started in current state")
	}

	if err := cntr.ExecHooks("startContainer"); err != nil {
		log.Error().Err(err).Msg("failed to execute startContainer hooks")
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	conn, err := net.Dial("unix", filepath.Join(cntr.State.Bundle, containerSockFilename))
	if err != nil {
		log.Error().Err(err).Msg("failed to dial container sockaddr")
		return fmt.Errorf("dial socket: %w", err)
	}

	if err := cntr.ExecHooks("prestart"); err != nil {
		log.Error().Err(err).Msg("failed to execute prestart hooks")
		cntr.State.Status = specs.StateStopped
		cntr.Save()
		log.Info().Msg("BEFORE FAIL DELETE")

		// TODO: run DELETE tasks here, then...

		log.Info().Msg("execing poststop hooks")
		if err := cntr.ExecHooks("poststop"); err != nil {
			fmt.Println("WARNING: failed to execute poststop hooks")
			log.Warn().Err(err).Msg("failed to execute poststop hooks")
		}

		return errors.New("failed to run prestart hooks")
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("failed to send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}
	defer conn.Close()

	// FIXME: ?? when process starts, the PID in state should be updated to the process IN the container??

	// cntr.State.Status = specs.StateStopped
	// if err := cntr.Save(); err != nil {
	// 	log.Error().Err(err).Msg("save state after stopped")
	// 	return fmt.Errorf("failed to save stopped state: %w", err)
	// }

	return nil
}
