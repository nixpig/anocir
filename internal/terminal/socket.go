package terminal

import (
	"fmt"
	"strconv"
	"syscall"
)

type PtySocket struct {
	FD   int
	Name string
}

func OpenPtySocket(fd int, path string) *PtySocket {
	return &PtySocket{
		FD:   fd,
		Name: path,
	}
}

func NewPtySocket(path string) (*PtySocket, error) {
	fd, err := syscall.Socket(
		syscall.AF_UNIX,
		syscall.SOCK_STREAM,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("create terminal socket: %w", err)
	}

	return &PtySocket{
		FD:   fd,
		Name: path,
	}, nil
}

func (p *PtySocket) Connect() error {
	if err := syscall.Connect(
		p.FD,
		&syscall.SockaddrUnix{
			Name: p.Name,
		},
	); err != nil {
		return fmt.Errorf("connect terminal socket: %w", err)
	}

	return nil
}

func (p *PtySocket) Close() error {
	if err := syscall.Close(p.FD); err != nil {
		return fmt.Errorf("close terminal socket: %w", err)
	}

	return nil
}

func (p *PtySocket) SendMsg(pty *Pty) error {
	if err := syscall.Sendmsg(
		p.FD,
		[]byte(strconv.Itoa(pty.MasterFD)),
		[]byte{syscall.SCM_RIGHTS},
		nil,
		0,
	); err != nil {
		return fmt.Errorf("terminal sendmsg: %w", err)
	}

	return nil
}
