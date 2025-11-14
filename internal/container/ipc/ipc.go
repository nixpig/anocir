package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
)

type Socket struct {
	path string
}

func NewSocket(path string) *Socket {
	return &Socket{path}
}

func (s *Socket) Listen() (net.Listener, error) {
	return net.Listen("unix", s.path)
}

func (s *Socket) Dial() (net.Conn, error) {
	return net.Dial("unix", s.path)
}

func (s *Socket) DialWithRetry() (net.Conn, error) {
	timeout := 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var conn net.Conn
	var err error

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to connect after %v", timeout)
		case <-ticker.C:
			conn, err = s.Dial()
			if err != nil {
				continue
			}

			return conn, nil
		}
	}
}

func (s *Socket) SetPermissions(mode os.FileMode) error {
	return os.Chmod(s.path, mode)
}

func SendMessage(conn net.Conn, msg string) error {
	_, err := conn.Write([]byte(msg))

	return err
}

func ReceiveMessage(conn net.Conn) (string, error) {
	buf := make([]byte, 128)

	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read message: %w", err)
	}

	return string(buf[:n]), nil
}
