// main.go

package main

import (
	"os"

	"github.com/nixpig/anocir/internal/cli"
)

func main() {
	if err := cli.RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
