package database

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/pkg"
)

type DB struct {
	Conn *sql.DB
}

func New() *DB {
	return &DB{}
}

func (d *DB) Connect() error {
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

func (d *DB) GetBundleFromID(id string) (string, error) {
	row := d.Conn.QueryRow(
		`select bundle_ from containers_ where id_ = $id`,
		sql.Named("id", id),
	)

	var bundle string
	if err := row.Scan(&bundle); err != nil {
		return "", err
	}

	return bundle, nil
}

func (d *DB) DeleteContainerByID(id string) error {
	res, err := d.Conn.Exec(
		`delete from containers_ where id_ = $id`,
		sql.Named("id", id),
	)
	if err != nil {
		return fmt.Errorf("delete container db: %w", err)
	}

	if c, err := res.RowsAffected(); err != nil || c == 0 {
		return errors.New("didn't delete container for whatever reason")
	}

	return nil
}

func (d *DB) CreateContainer(id, bundle string) error {
	query := `insert into containers_ (
		id_, bundle_
	) values (
		$id, $bundle
	)`
	if _, err := d.Conn.Exec(
		query,
		sql.Named("id", id),
		sql.Named("bundle", bundle),
	); err != nil {
		return fmt.Errorf("insert into db: %w", err)
	}

	return nil
}
