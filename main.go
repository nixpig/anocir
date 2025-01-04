// main.go

package main

import (
	"fmt"
	"os"

	"github.com/nixpig/anocir/internal/cli"
)

func main() {
	if err := cli.RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
