package cmd

import "fmt"

func Start(containerID string) error {
	fmt.Println("start container: ")
	fmt.Println("containerID: ", containerID)
	return nil
}
