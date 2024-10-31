package main

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/nixpig/brownie/cli"
	"github.com/nixpig/brownie/internal/database"
	"github.com/nixpig/brownie/internal/logging"
	"github.com/nixpig/brownie/pkg"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var mig embed.FS

func main() {
	// create logger
	logPath := filepath.Join(pkg.BrownieRootDir, "logs", "brownie.log")
	log, err := logging.CreateLogger(logPath, zerolog.InfoLevel)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// create database
	db := database.New()
	if err := db.Connect(); err != nil {
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
	if err := cli.RootCmd(log, db).Execute(); err != nil {
		log.Error().Err(err).Msg("failed to exec cmd")
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
