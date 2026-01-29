package main

import (
	"os"

	_ "github.com/nixpig/anocir/internal/nssetup"
	cli "github.com/nixpig/anocir/internal/oci"
)

func main() {
<<<<<<< HEAD
=======
	if err := gons.Status(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to join namespaces: %s\n", err)
		os.Exit(1)
	}

>>>>>>> c3e448a2705510b9a6826ea40a8539d19c64960a
	if err := cli.RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
