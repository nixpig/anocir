package commands

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/container"
	"github.com/rs/zerolog"
)

func Delete(opts *container.DeleteOpts, log *zerolog.Logger, db *sql.DB) error {
	cntr, err := container.Load(opts.ID, log, db)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	return cntr.Delete(opts, log, db)
}
