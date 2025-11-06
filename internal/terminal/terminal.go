// Package terminal provides functionality for managing pseudo-terminals (PTYs)
// and console sockets for container processes.
package terminal

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/google/goterm/term"
	"golang.org/x/sys/unix"
)

// Pty represents a pseudo-terminal pair, consisting of a master and a slave
// file.
type Pty struct {
	Master *os.File
	Slave  *os.File
}

// NewPty creates a new Pty pseudo-terminal pair.
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

// Connect sets up the Pty slave as the controlling terminal and redirects
// stdin, stdout, and stderr to it.
func (p *Pty) Connect() error {
	if _, err := unix.Setsid(); err != nil {
		return fmt.Errorf("setsid: %w", err)
	}

	if err := unix.IoctlSetInt(int(p.Slave.Fd()), syscall.TIOCSCTTY, 0); err != nil {
		return fmt.Errorf("set ioctl: %w", err)
	}

	if err := syscall.Dup2(int(p.Slave.Fd()), 0); err != nil {
		return fmt.Errorf("dup2 stdin: %w", err)
	}

	if err := syscall.Dup2(int(p.Slave.Fd()), 1); err != nil {
		return fmt.Errorf("dup2 stdout: %w", err)
	}

	if err := syscall.Dup2(int(p.Slave.Fd()), 2); err != nil {
		return fmt.Errorf("dup2 stderr: %w", err)
	}

	return nil
}

// MountSlave mounts the Pty slave device to the specified target path.
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

	if err := syscall.Mount(
		p.Slave.Name(),
		target,
		"bind",
		syscall.MS_BIND,
		"",
	); err != nil {
		return fmt.Errorf(
			"mount pty slave device (%s) to target (%s): %w",
			p.Slave.Name(),
			target,
			err,
		)
	}

	return nil
}

// PtySocket represents a Unix domain socket used for communicating with a Pty.
type PtySocket struct {
	SocketFd int
}

// NewPtySocket creates a new PtySocket and connects it at the specified path.
func NewPtySocket(consoleSocketPath string) (*PtySocket, error) {
	fd, err := syscall.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, fmt.Errorf("create console socket: %w", err)
	}

	if err := syscall.Connect(
		fd,
		&syscall.SockaddrUnix{
			Name: consoleSocketPath,
		},
	); err != nil {
		return nil, fmt.Errorf("connect to console socket: %w", err)
	}

	return &PtySocket{
		SocketFd: fd,
	}, nil
}

// Close closes the PtySocket.
func (ps *PtySocket) Close() error {
	if err := syscall.Close(ps.SocketFd); err != nil {
		return fmt.Errorf("close console socket: %w", err)
	}

	return nil
}

// SendPty sends the master file descriptor of a Pty over a Unix domain socket.
func SendPty(consoleSocket int, pty *Pty) error {
	masterFds := []int{int(pty.Master.Fd())}
	cmsg := syscall.UnixRights(masterFds...)
	size := unsafe.Sizeof(pty.Master.Fd())
	buf := make([]byte, size)

	switch size {
	case 4:
		binary.NativeEndian.PutUint32(buf, uint32(pty.Master.Fd()))
	case 8:
		binary.NativeEndian.PutUint64(buf, uint64(pty.Master.Fd()))
	default:
		return fmt.Errorf("unsupported architecture (%d)", size*8)
	}

	if err := syscall.Sendmsg(consoleSocket, buf, cmsg, nil, 0); err != nil {
		return fmt.Errorf("terminal sendmsg: %w", err)
	}

	return nil
}

// Setup prepares the console for the container process. It changes the current
// working directory to the rootfs, creates a symlink for the console socket,
// and returns the file descriptor of the console socket.
func Setup(rootfs, consoleSocketPath string) (*int, error) {
	consoleSocketSymlink := filepath.Join(rootfs, "console-socket")

	if err := os.Symlink(consoleSocketPath, consoleSocketSymlink); err != nil {
		return nil, fmt.Errorf("symlink console socket: %w", err)
	}

	consoleSocket, err := NewPtySocket(consoleSocketSymlink)
	if err != nil {
		return nil, fmt.Errorf("create terminal socket: %w", err)
	}

	return &consoleSocket.SocketFd, nil
}
