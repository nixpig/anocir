package main

import (
	"fmt"
	"os"

	"github.com/nixpig/brownie/internal/cli"
	"github.com/thediveo/gons"
)

func main() {
	// check namespace status
	if err := gons.Status(); err != nil {
		fmt.Println("join namespace(s): ", err)
		os.Exit(1)
	}

	// exec root
	if err := cli.RootCmd().Execute(); err != nil {
		fmt.Println(fmt.Errorf("%s, %w", os.Args, err))
		os.Exit(1)
	}

	os.Exit(0)
}
