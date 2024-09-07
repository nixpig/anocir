package cmd

import "fmt"

func State(containerID string) error {
	fmt.Println("query state: ", containerID)
	return nil
}
