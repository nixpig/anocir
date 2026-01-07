// Package ipc provides functionality for inter-process communication between a
// forked container process and the runtime.
package ipc

// TODO: Now it's clear exactly what IPC is required, this could probably be
// simplified significantly by using EventFD or some other
// condition-variable-like mechanism to 'nofify' on "ready" and "start".

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// MsgStart is the message sent on the container socket to start the created
	// container.
	MsgStart byte = iota + 1
	// MsgReady is the message sent over the init socketpair when the container
	// is created and ready to receive commands.
	MsgReady
	// MsgInvalidBinary is the message sent over the init socketpair when the exec
	// binary cannot be found.
	MsgInvalidBinary
	MsgPrePivot
)

// Socket holds a path to use for a unix domain socket.
type Socket struct {
	path string
}

// NewSocket creates a Socket with the given path.
func NewSocket(path string) *Socket {
	return &Socket{path}
}

// Listen returns a listener on the Socket path.
func (s *Socket) Listen() (net.Listener, error) {
	return net.Listen("unix", s.path)
}

// Dial returns a connection to the Socket path.
func (s *Socket) Dial() (net.Conn, error) {
	return net.Dial("unix", s.path)
}

// DialWithRetry attempts to dial the Socket path, retrying at the given
// interval until a connection is established and returns the connection or the
// given timeout is reached and returns an error.
func (s *Socket) DialWithRetry(
	interval, timeout time.Duration,
) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		timeout*time.Millisecond,
	)
	defer cancel()

	ticker := time.NewTicker(interval * time.Millisecond)
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

// SetPermissions sets the given mode on the Socket path.
func (s *Socket) SetPermissions(mode os.FileMode) error {
	return os.Chmod(s.path, mode)
}

// FDToConn returns a FileConn to the given fd.
func FDToConn(fd int) (net.Conn, error) {
	return net.FileConn(os.NewFile(uintptr(fd), "ipc_socket"))
}

// SendMessage writes the given msg to the given conn.
func SendMessage(conn net.Conn, msg byte) error {
	_, err := conn.Write([]byte{msg})

	return err
}

// ReceiveMessage reads from the given conn and returns the read data.
func ReceiveMessage(conn net.Conn) (byte, error) {
	buf := make([]byte, 1)
	_, err := conn.Read(buf)

	return buf[0], err
}

// NewSocketPair creates a socket pair and returns the file descriptors.
func NewSocketPair() (int, int, error) {
	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM|unix.SOCK_CLOEXEC,
		0,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("new socket pair: %w", err)
	}

	return fds[0], fds[1], nil
}
