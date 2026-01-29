package main

import (
	"os"

	_ "github.com/nixpig/anocir/internal/nssetup"
	cli "github.com/nixpig/anocir/internal/oci"
)

func main() {
	if err := cli.RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
