package terminal

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

type Pty struct {
	MasterFD int
	SlaveFD  int
}

func NewPty() (*Pty, error) {
	masterFD, err := syscall.Open(
		"/dev/ptmx",
		syscall.O_NOCTTY|syscall.O_RDWR|syscall.O_CLOEXEC,
		0666,
	)
	if err != nil {
		return nil, err
	}

	if err := unix.IoctlSetInt(masterFD, unix.TIOCGPTN, 0); err != nil {
		return nil, err
	}

	if err := unix.IoctlSetInt(masterFD, unix.TIOCSPTLCK, 0); err != nil {
		return nil, err
	}

	slavePtyFD, err := unix.IoctlGetInt(masterFD, unix.TIOCGPTN)
	if err != nil {
		return nil, err
	}
	slavePts := fmt.Sprintf("/dev/pts/%d", slavePtyFD)

	slaveFD, err := syscall.Open(slavePts, syscall.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("open slave pts: %w", err)
	}

	return &Pty{
		MasterFD: masterFD,
		SlaveFD:  slaveFD,
	}, nil
}

func (p *Pty) Connect() error {
	if err := syscall.Dup2(p.SlaveFD, 0); err != nil {
		return err
	}

	if err := syscall.Dup2(p.SlaveFD, 1); err != nil {
		return err
	}

	if err := syscall.Dup2(p.SlaveFD, 2); err != nil {
		return err
	}

	return nil
}

func (p *Pty) Close() error {
	if err := syscall.Close(p.SlaveFD); err != nil {
		return fmt.Errorf("close slave pty: %w", err)
	}

	if err := syscall.Close(p.MasterFD); err != nil {
		return fmt.Errorf("close master pty: %w", err)
	}

	return nil
}
