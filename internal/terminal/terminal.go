// Package terminal provides functionality for managing pseudo-terminals (PTYs)
// and console sockets for container processes.
package terminal

import (
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"

	"github.com/google/goterm/term"
	"github.com/nixpig/anocir/internal/platform"
	"golang.org/x/sys/unix"
)

// Pty represents a pseudo-terminal pair, consisting of a Master and a Slave
// File.
type Pty struct {
	// Master is the master side of the pseudo-terminal pair held by the runtime
	// process.
	Master *os.File
	// Slave is the slave side of the pseudo-terminal pair used as the
	// controlling terminal for the container process.
	Slave *os.File
}

// NewPty creates a Pty pseudo-terminal pair.
func NewPty() (*Pty, error) {
	pty, err := term.OpenPTY()
	if err != nil {
		return nil, fmt.Errorf("open pty: %w", err)
	}

	return &Pty{
		Master: pty.Master,
		Slave:  pty.Slave,
	}, nil
}

// Connect sets up the Pty Slave as the controlling terminal and redirects
// stdin, stdout, and stderr to it.
func (p *Pty) Connect() error {
	if _, err := unix.Setsid(); err != nil {
		return fmt.Errorf("setsid: %w", err)
	}

	if err := unix.IoctlSetInt(int(p.Slave.Fd()), unix.TIOCSCTTY, 0); err != nil {
		return fmt.Errorf("set ioctl: %w", err)
	}

	if err := unix.Dup2(int(p.Slave.Fd()), 0); err != nil {
		return fmt.Errorf("dup2 stdin: %w", err)
	}

	if err := unix.Dup2(int(p.Slave.Fd()), 1); err != nil {
		return fmt.Errorf("dup2 stdout: %w", err)
	}

	if err := unix.Dup2(int(p.Slave.Fd()), 2); err != nil {
		return fmt.Errorf("dup2 stderr: %w", err)
	}

	return nil
}

// MountSlave mounts the Pty Slave device to the specified target path.
func (p *Pty) MountSlave(target string) error {
	if _, err := os.Stat(target); os.IsNotExist(err) {
		f, err := os.Create(target)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("create device target if not exists: %w", err)
		}
		if f != nil {
			f.Close()
		}
	}

	if err := platform.BindMount(p.Slave.Name(), target, false); err != nil {
		return fmt.Errorf("bind mount pty slave device: %w", err)
	}

	return nil
}

// PtySocket represents a unix domain socket used for communicating with a Pty.
type PtySocket struct {
	// SocketFd is the file descriptor to use for the unix domain socket.
	SocketFd int
}

// NewPtySocket creates a new PtySocket and connects it at the specified
// consoleSocketPath.
func NewPtySocket(consoleSocketPath string) (*PtySocket, error) {
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("create console socket: %w", err)
	}

	if err := unix.Connect(
		fd,
		&unix.SockaddrUnix{
			Name: consoleSocketPath,
		},
	); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("connect to console socket: %w", err)
	}

	return &PtySocket{
		SocketFd: fd,
	}, nil
}

// Close closes the PtySocket.
func (ps *PtySocket) Close() error {
	if err := unix.Close(ps.SocketFd); err != nil {
		return fmt.Errorf("close console socket: %w", err)
	}

	return nil
}

// SendPty sends the Master file descriptor of a Pty over a unix domain socket.
func SendPty(consoleSocket int, pty *Pty) error {
	masterFds := []int{int(pty.Master.Fd())}
	cmsg := unix.UnixRights(masterFds...)
	size := unsafe.Sizeof(pty.Master.Fd())
	buf := make([]byte, size)

	// Ensure FD number is encoded correctly for the architecture.
	switch size {
	case 4:
		binary.NativeEndian.PutUint32(buf, uint32(pty.Master.Fd()))
	case 8:
		binary.NativeEndian.PutUint64(buf, uint64(pty.Master.Fd()))
	default:
		return fmt.Errorf("unsupported architecture (%d)", size*8)
	}

	if err := unix.Sendmsg(consoleSocket, buf, cmsg, nil, 0); err != nil {
		return fmt.Errorf("terminal sendmsg: %w", err)
	}

	return nil
}

// Setup prepares the console for the container process. It changes the current
// working directory to the given rootfs, creates a symlink for the console
// socket, and returns the file descriptor of the console socket at
// consoleSocketPath.
func Setup(rootfs, consoleSocketPath string) (int, error) {
	consoleSocket, err := NewPtySocket(consoleSocketPath)
	if err != nil {
		return 0, fmt.Errorf("create terminal socket: %w", err)
	}

	return consoleSocket.SocketFd, nil
}
