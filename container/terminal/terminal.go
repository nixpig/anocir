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
	master *os.File
	slave  *os.File
}

func NewPty() (*Pty, error) {
	pty, err := term.OpenPTY()
	if err != nil {
		return nil, fmt.Errorf("open pty: %w", err)
	}

	master, slave := pty.Master, pty.Slave

	return &Pty{
		master: master,
		slave:  slave,
	}, nil
}

func (p *Pty) Connect() error {
	if _, err := unix.Setsid(); err != nil {
		return fmt.Errorf("setsid: %w", err)
	}

	if err := syscall.Dup2(int(p.slave.Fd()), 0); err != nil {
		return fmt.Errorf("dup2 stdin: %w", err)
	}

	if err := syscall.Dup2(int(p.slave.Fd()), 1); err != nil {
		return fmt.Errorf("dup2 stdout: %w", err)
	}

	if err := syscall.Dup2(int(p.slave.Fd()), 2); err != nil {
		return fmt.Errorf("dup2 stderr: %w", err)
	}

	return nil
}

type PtySocket struct {
	SocketFd int
}

func NewPtySocket(consoleSocketPath string) (*PtySocket, error) {
	fd, err := syscall.Socket(
		syscall.AF_UNIX,
		syscall.SOCK_STREAM,
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
	masterFds := []int{int(pty.master.Fd())}

	cmsg := syscall.UnixRights(masterFds...)

	size := unsafe.Sizeof(pty.master.Fd())

	buf := make([]byte, size)

	switch size {
	case 4:
		binary.NativeEndian.PutUint32(buf, uint32(pty.master.Fd()))
	case 8:
		binary.NativeEndian.PutUint64(buf, uint64(pty.master.Fd()))
	default:
		panic("done fucked up")
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
