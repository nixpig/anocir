package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
)

func server(c net.Conn) error {
	defer c.Close()
	fmt.Println("client connected: ", c.RemoteAddr().Network())
	if _, err := io.Copy(c, c); err != nil {
		return fmt.Errorf("copy from/to connection: %w", err)
	}
	fmt.Println("after copy")

	return nil
}

func Fork(forkedCmd, containerID, bundlePath string) error {
	fmt.Println("forkedCmd: ", forkedCmd)
	fmt.Println("containerID: ", containerID)
	fmt.Println("bundlePath: ", bundlePath)

	sockAddr := fmt.Sprintf("/tmp/brownie_%s.sock", containerID)

	if err := os.RemoveAll(sockAddr); err != nil {
		return fmt.Errorf("remove existing socket: %w", err)
	}

	l, err := net.Listen("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
			continue
		}

		go server(conn)
	}
}
