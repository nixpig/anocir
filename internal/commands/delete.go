package commands

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/internal/container"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts, log *zerolog.Logger, db *sql.DB) error {
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if !opts.Force && !cntr.CanBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state: %s", cntr.State.Status)
	}

	process, _ := os.FindProcess(cntr.State.PID)
	if process != nil {
		process.Signal(syscall.Signal(9))
	}

	res, err := db.Exec(`delete from containers_ where id_ = $id`, sql.Named("id", opts.ID))
	if err != nil {
		return fmt.Errorf("delete container db: %w", err)
	}

	if c, err := res.RowsAffected(); err != nil || c == 0 {
		return errors.New("didn't delete container for whatever reason")
	}

	log.Info().Msg("execing poststop hooks")
	if err := cntr.ExecHooks("poststop"); err != nil {
		log.Warn().Err(err).Msg("failed to execute poststop hooks")
	}

	return nil
}
