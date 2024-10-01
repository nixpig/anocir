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
	err := os.MkdirAll(filepath.Join(pkg.BrownieRootDir, "logs"), 0666)
	if err != nil {
		fmt.Println("create log dir: %w", err)
		os.Exit(1)
	}

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

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
