package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/pkg"
)

type db struct {
	Conn *sql.DB
}

func New() *db {
	return &db{}
}

func (d *db) Connect() error {
	databaseConnectionString := fmt.Sprintf(
		"file:%s?_auth&_auth_user=%s&_auth_pass=%s&_auth_crypt=sha1",
		filepath.Join(pkg.BrownieRootDir, "containers.db"),
		// FIXME: pull user/password from env variables
		"user",
		"password",
	)

	db, err := sql.Open("sqlite3", databaseConnectionString)
	if err != nil {
		return fmt.Errorf("failed to open database file (%s): %w", databaseConnectionString, err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database (%s): %w", databaseConnectionString, err)
	}

	d.Conn = db

	return nil
}
