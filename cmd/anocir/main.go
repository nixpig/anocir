// main.go

package main

import (
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/cli"
	"github.com/thediveo/gons"
)

func main() {
	if err := gons.Status(); err != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("failed to join namespaces: %s\n", err)))
		os.Exit(1)
	}

	if err := cli.RootCmd().Execute(); err != nil {
		os.Stderr.Write([]byte(fmt.Sprintf("failed to execute: %s\n", err)))
		os.Exit(1)
	}
}
