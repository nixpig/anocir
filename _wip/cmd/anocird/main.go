package main

import (
	"os"

	"github.com/nixpig/anocir/internal/cri"
)

func main() {
	if err := cri.Cmd().Execute(); err != nil {
		os.Exit(1)
	}
}
