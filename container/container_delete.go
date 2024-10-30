package container

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func (c *Container) Delete(opts *DeleteOpts, log *zerolog.Logger, db *sql.DB) error {
	if !opts.Force && !c.CanBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state: %s", c.State.Status)
	}

	process, err := os.FindProcess(c.State.PID)
	if err != nil {
		return fmt.Errorf("find container process: %w", err)
	}
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

	if err := c.ExecHooks("poststop"); err != nil {
		fmt.Println("failed to execute poststop hooks")
		log.Warn().Err(err).Msg("failed to execute poststop hooks")
	}

	return nil
}
