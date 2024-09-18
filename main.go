package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/cmd"
	"github.com/nixpig/brownie/pkg"
	"github.com/rs/zerolog"
)

func main() {
	logfile, err := os.OpenFile(
		filepath.Join(pkg.BrownieRootDir, "logs", "brownie.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		fmt.Println("open log file: %w", err)
		os.Exit(1)
	}

	log := zerolog.New(logfile).With().Timestamp().Logger()

	rootCmd := cmd.RootCmd(&log)

	if os.Args[1] == "create" {

		os.Stdout.Write([]byte(" >>> CREATED 1"))
		rootCmd.OutOrStdout().Write([]byte(" >>> CREATED 2"))
	}
	if os.Args[1] == "start" {
		os.Stdout.Write([]byte(" >>> STARTED 1"))
		rootCmd.OutOrStdout().Write([]byte(" >>> STARTED 2"))
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
