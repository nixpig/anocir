package terminal

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/google/goterm/term"
	"golang.org/x/sys/unix"
)

type Pty struct {
	Master *os.File
	Slave  *os.File
}

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
		return fmt.Errorf("mount pty slave device (%s) to target (%s): %w", p.Slave.Name(), target, err)
	}

	return nil
}

type PtySocket struct {
	SocketFd int
}

func NewPtySocket(consoleSocketPath string) (*PtySocket, error) {
	fd, err := syscall.Socket(
		unix.AF_UNIX,
		unix.SOCK_STREAM,
		0,
	)
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

func (ps *PtySocket) Close() error {
	if err := syscall.Close(ps.SocketFd); err != nil {
		return fmt.Errorf("close console socket: %w", err)
	}

	return nil
}

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

	if err := syscall.Sendmsg(
		consoleSocket,
		buf,
		cmsg,
		nil,
		0,
	); err != nil {
		return fmt.Errorf("terminal sendmsg: %w", err)
	}

	return nil
}

func Setup(rootfs, consoleSocketPath string) (*int, error) {
	prev, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get cwd: %w", err)
	}

	if err := os.Chdir(rootfs); err != nil {
		return nil, fmt.Errorf("change to container root dir: %w", err)
	}

	if err := os.Symlink(consoleSocketPath, "./console-socket"); err != nil {
		return nil, fmt.Errorf("symlink console socket: %w", err)
	}

	consoleSocket, err := NewPtySocket(
		"./console-socket",
	)
	if err != nil {
		return nil, fmt.Errorf("create terminal socket: %w", err)
	}

	if err := os.Chdir(prev); err != nil {
		return nil, fmt.Errorf("change back to previos cwd: %w", err)
	}

	return &consoleSocket.SocketFd, nil
}
