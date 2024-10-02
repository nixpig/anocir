package main

import (
	"fmt"
	"os"

	"github.com/nixpig/brownie/cmd"
)

func main() {
	if err := cmd.RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
