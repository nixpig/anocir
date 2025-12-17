package terminal_test

import (
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/nixpig/anocir/internal/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestNewPty(t *testing.T) {
	pty, err := terminal.NewPty()
	require.NoError(t, err)

	defer func() {
		pty.Master.Close()
		pty.Slave.Close()
	}()

	assert.Equal(t, "/dev/ptmx", pty.Master.Name())
	assert.Regexp(t, `^/dev/pts/\d+$`, pty.Slave.Name())

	_, err = pty.Master.Stat()
	require.NoError(t, err)

	_, err = pty.Slave.Stat()
	require.NoError(t, err)

	_, err = pty.Slave.WriteString("test")
	require.NoError(t, err)

	buf := make([]byte, 4)
	n, err := pty.Master.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "test", string(buf[:n]))
}

func TestNewPtySocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "test.sock")

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	ptySocket, err := terminal.NewPtySocket(socketPath)
	require.NoError(t, err)
	defer ptySocket.Close()

	assert.NoError(t, unix.Fstat(ptySocket.SocketFd, &unix.Stat_t{}))
}

func TestNewPtySocket_NonExistentPath(t *testing.T) {
	ptySocket, err := terminal.NewPtySocket("/nonexistent/path.sock")
	assert.Error(t, err)
	assert.Nil(t, ptySocket)
}

func TestSendPty(t *testing.T) {
	pty, err := terminal.NewPty()
	require.NoError(t, err)

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	ptySocket, err := terminal.NewPtySocket(socketPath)
	require.NoError(t, err)
	ptySocket.Close()

	assert.Error(t, terminal.SendPty(ptySocket.SocketFd, pty))
}

func TestPtyE2E(t *testing.T) {
	pty, err := terminal.NewPty()
	require.NoError(t, err)

	defer func() {
		pty.Master.Close()
		pty.Slave.Close()
	}()

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	fdCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer conn.Close()

		unixConn := conn.(*net.UnixConn)

		buf := make([]byte, 32)
		oob := make([]byte, unix.CmsgSpace(4))

		_, oobn, _, _, err := unixConn.ReadMsgUnix(buf, oob)
		if err != nil {
			errCh <- err
			return
		}

		msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			errCh <- err
			return
		}
		if len(msgs) == 0 {
			errCh <- errors.New("messages length is zero")
			return
		}

		fds, err := unix.ParseUnixRights(&msgs[0])
		if err != nil {
			errCh <- err
			return
		}
		if len(fds) == 0 {
			errCh <- errors.New("fds length is zero")
			return
		}

		fdCh <- fds[0]
	}()

	ptySocket, err := terminal.NewPtySocket(socketPath)
	require.NoError(t, err)
	defer ptySocket.Close()

	require.NoError(t, terminal.SendPty(ptySocket.SocketFd, pty))

	select {
	case fd := <-fdCh:
		defer unix.Close(fd)

		_, err = pty.Slave.WriteString("test")
		require.NoError(t, err)

		buf := make([]byte, 64)
		n, err := unix.Read(fd, buf)
		require.NoError(t, err)
		assert.Equal(t, "test", string(buf[:n]))
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "timed out waiting for file descriptors")
	}
}

func TestSendPty_InvalidSocket(t *testing.T) {
	pty, err := terminal.NewPty()
	require.NoError(t, err)

	assert.Error(t, terminal.SendPty(-1, pty))
}
