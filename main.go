package main

import (
	"os"

	"github.com/nixpig/anocir/internal/cli"
	_ "github.com/nixpig/anocir/internal/nssetup"
)

func main() {
	if err := cli.RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
