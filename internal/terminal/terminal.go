package terminal

import "fmt"

type Terminal struct {
	FD int
}

func New(consoleSockPath string) (*Terminal, error) {
	ptySock, err := NewPtySocket(consoleSockPath)
	if err != nil {
		return nil, fmt.Errorf("create new pty sock: %w", err)
	}

	if err := ptySock.Connect(); err != nil {
		return nil, fmt.Errorf("connect to pty sock: %w", err)
	}

	return &Terminal{FD: ptySock.FD}, nil
}
