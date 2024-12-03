package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal/cli"
	"github.com/nixpig/brownie/internal/logging"
	"github.com/rs/zerolog"
	"github.com/thediveo/gons"
)

const (
	brownieRootDir = "/var/lib/brownie"
)

func main() {
	// check namespace status
	if err := gons.Status(); err != nil {
		fmt.Println("join namespace(s): ", err)
		os.Exit(1)
	}

	// create logger
	logPath := filepath.Join(brownieRootDir, "logs", "brownie.log")
	log, err := logging.CreateLogger(logPath, zerolog.InfoLevel)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// exec root
	log.Info().Any("args", os.Args[1:]).Msg("COMMAND")
	if err := cli.RootCmd(log, logPath).Execute(); err != nil {
		log.Error().Err(err).Str("cmd", os.Args[1]).Msg("failed to exec cmd")
		fmt.Println(fmt.Errorf("ERROR: %s, %w", os.Args, err))
		os.Exit(1)
	}

	os.Exit(0)
}
