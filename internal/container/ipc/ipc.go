// Package ipc provides functionality for inter-process communication between a
// forked container process and the runtime.
package ipc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	// MsgPrePivot is the message sent before pivot_root is called.
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
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
func NewSocketPair() (*os.File, *os.File, error) {
	fds, err := unix.Socketpair(
		unix.AF_UNIX,
		unix.SOCK_STREAM|unix.SOCK_CLOEXEC,
		0,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("new socket pair: %w", err)
	}

	parent := os.NewFile(uintptr(fds[0]), "ipc_sock_parent")
	child := os.NewFile(uintptr(fds[1]), "ipc_sock_child")

	return parent, child, nil
}

// ShortID constructs a hash of the given bundle. It's used to create the
// directory for storing IPC socket files.
func ShortID(bundle string) string {
	hash := sha256.Sum256([]byte(bundle))
	return hex.EncodeToString(hash[:8])
}
