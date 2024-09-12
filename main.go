package main

import (
	"os"

	"github.com/nixpig/brownie/cmd"
)

func main() {
	if err := cmd.Root.Execute(); err != nil {
		os.Exit(1)
	}
}
