package main

import (
	"fmt"
	"os"

	cli "github.com/nixpig/anocir/internal/oci"
	"github.com/thediveo/gons"
)

func main() {
	if err := gons.Status(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to join namespaces: %s\n", err)
		os.Exit(1)
	}

	if err := cli.RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
