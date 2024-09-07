package cmd

import "fmt"

func Create(containerID, bundlePath string) error {
	fmt.Println("create container: ")
	fmt.Println("containerID: ", containerID)
	fmt.Println("bundlePath: ", bundlePath)
	return nil
}
