package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nixpig/brownie/cmd"
)

func main() {
	var err error

	argsLength := len(os.Args)

	if argsLength < 2 {
		fmt.Println("no command provided")
		os.Exit(1)
	}

	subcmd := os.Args[1]

	switch subcmd {
	case "state":
		if argsLength < 3 {
			err = errors.New("missing arguments - expected <container-id>")
			break
		}

		containerID := os.Args[2]
		err = cmd.State(containerID)

	case "create":
		if argsLength < 4 {
			err = errors.New("missing arguments - expected <container-id> <path-to-bundle>")
			break
		}
		containerID := os.Args[2]
		bundlePath := os.Args[3]
		err = cmd.Create(containerID, bundlePath)

	case "start":
		if argsLength < 3 {
			err = errors.New("missing arguments - expected <container-id>")
			break
		}
		containerID := os.Args[2]
		err = cmd.Start(containerID)

	case "kill":
		if argsLength < 4 {
			err = errors.New("missing arguments - expected <container-id> <signal>")
			break
		}
		containerID := os.Args[2]
		signal := os.Args[3]
		err = cmd.Kill(containerID, signal)

	case "delete":
		if argsLength < 3 {
			err = errors.New("missing arguments - expected <container-id>")
			break
		}
		containerID := os.Args[2]
		cmd.Delete(containerID)

	default:
		err = errors.New("unknown command")
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
