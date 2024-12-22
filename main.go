package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/nixpig/brownie/internal/cli"
	"github.com/thediveo/gons"
)

func main() {
	log, err := os.OpenFile("/var/log/brownie.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("open log file: ", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(io.MultiWriter(log, os.Stdout), nil))
	slog.SetDefault(logger)

	if err := gons.Status(); err != nil {
		slog.Error("join namespace(s)", slog.String("err", err.Error()))
		os.Exit(1)
	}

	if err := cli.RootCmd().Execute(); err != nil {
		slog.Error("root exec", slog.String("err", err.Error()))
		os.Exit(1)
	}

	os.Exit(0)
}
