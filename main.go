package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/nixpig/brownie/internal/cli"
	"github.com/nixpig/brownie/internal/database"
	"github.com/nixpig/brownie/internal/logging"
	"github.com/rs/zerolog"
)

const (
	brownieRootDir = "/var/lib/brownie"
)

//go:embed migrations/*.sql
var mig embed.FS

func main() {
	// create logger
	logPath := filepath.Join(brownieRootDir, "logs", "brownie.log")
	log, err := logging.CreateLogger(logPath, zerolog.InfoLevel)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// create database
	db := database.New()
	if err := db.Connect(fmt.Sprintf(
		"file:%s?_auth&_auth_user=%s&_auth_pass=%s&_auth_crypt=sha1",
		filepath.Join(brownieRootDir, "containers.db"),
		// FIXME: pull user/password from env variables
		"user",
		"password",
	)); err != nil {
		log.Error().Err(err).Msg("failed to connect to database")
		fmt.Println(err)
		os.Exit(1)
	}

	migrations, err := iofs.New(mig, "migrations")
	if err != nil {
		log.Error().Err(err).Msg("failed to load migrations")
		fmt.Println(err)
		os.Exit(1)
	}

	migrator, err := database.NewMigration(
		db.Conn,
		migrations,
	)

	if err != nil {
		log.Error().Err(err).Msg("failed to create migrations")
		fmt.Println(err)
		os.Exit(1)
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error().Err(err).Msg("failed to run migrations")
		fmt.Println(err)
		os.Exit(1)
	}

	// exec root
	if err := cli.RootCmd(log, db, logPath).Execute(); err != nil {
		log.Error().Err(err).Msg("failed to exec cmd")
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
