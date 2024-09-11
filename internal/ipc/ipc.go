package ipc

import (
	"fmt"
	"net"
)

type IPCChild struct {
	listener net.Listener
}

type IPCChildListener interface {
	Listen() error
}

type IPCHandler = func(conn net.Conn, msg string) error

func NewChild(sockAddr string) (*IPCChild, error) {
	listener, err := net.Listen("unix", sockAddr)
	if err != nil {
		return nil, fmt.Errorf("create unix socket: %w", err)
	}

	return &IPCChild{listener}, nil
}

func (c *IPCChild) Listen(handle IPCHandler) error {
	defer c.listener.Close()

	for {
		conn, err := c.listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}

		go func() {
			defer conn.Close()

			b := make([]byte, 128)

			for {
				n, err := conn.Read(b)
				if err != nil {
					fmt.Println(err)
					break
				}

				if n == 0 {
					fmt.Println("no message")
					break
				}

				if err := handle(conn, string(b[:n])); err != nil {
					fmt.Println(err)
					if _, err := conn.Write([]byte(err.Error())); err != nil {
						fmt.Println(err)
					}
					break
				}
			}
		}()
	}
}
