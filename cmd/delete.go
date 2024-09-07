package cmd

import "fmt"

func Delete(containerID string) error {
	fmt.Println("delete container: ")
	fmt.Println("containerID: ", containerID)
	return nil
}
