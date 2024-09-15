package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	log.Info().Str("args", strings.Join(os.Args, ", ")).Msg("exec")

	if err := cmd.Root.Execute(); err != nil {
		fmt.Println("main: ", err)
		os.Exit(1)
	}

	log.Info().Msg("EXECUTED!")
}
