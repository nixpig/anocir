package cmd

import "fmt"

func Kill(containerID, signal string) error {
	fmt.Println("kill container: ")
	fmt.Println("containerID: ", containerID)
	fmt.Println("signal: ", signal)
	return nil
}
