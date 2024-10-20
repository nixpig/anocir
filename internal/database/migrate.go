package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/mattn/go-sqlite3"
)

type Migrator interface {
	Up() error
	Down() error
}

type Migration struct {
	migrate *migrate.Migrate
}

func (m Migration) Up() error {
	return m.migrate.Up()
}

func (m Migration) Down() error {
	return m.migrate.Down()
}

func NewMigration(db *sql.DB, migrations source.Driver) (*Migration, error) {
	instance, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance(
		"file",
		migrations,
		"sqlite3",
		instance,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new sqlite3 migrate instance: %w", err)
	}

	return &Migration{migrate: m}, nil
}
