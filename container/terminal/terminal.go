package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/goterm/term"
)

type Terminal struct {
	FD int
}

func SetupConsoleSocket(
	containerDir,
	consoleSocketPath,
	socketName string,
) (*int, error) {
	if err := os.Symlink(consoleSocketPath, filepath.Join(containerDir, socketName)); err != nil {
		return nil, fmt.Errorf("create socket symlink: %w", err)
	}

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
			Name: filepath.Join(containerDir, socketName),
		},
	); err != nil {
		return nil, fmt.Errorf("connect to console socket: %w", err)
	}

	return &fd, nil
}

func SetupConsole(fd int) error {
	pty, err := term.OpenPTY()
	if err != nil {
		return fmt.Errorf("open pty: %w", err)
	}

	master, slave := pty.Master, pty.Slave

	fds := []int{int(master.Fd())}
	rights := syscall.UnixRights(fds...)

	if err := syscall.Sendmsg(
		fd,
		[]byte("/dev/ptmx"),
		rights,
		nil,
		0,
	); err != nil {
		return fmt.Errorf("terminal sendmsg: %w", err)
	}

	if err := connectStdio(int(slave.Fd()), int(slave.Fd()), int(slave.Fd())); err != nil {
		return fmt.Errorf("connect to slave: %w", err)
	}

	if err := syscall.Close(fd); err != nil {
		return fmt.Errorf("close console socket: %w", err)
	}

	return nil
}

func connectStdio(stdin, stdout, stderr int) error {
	if err := syscall.Dup2(stdin, 0); err != nil {
		return fmt.Errorf("dup2 stdin from %d: %w", stdin, err)
	}

	if err := syscall.Dup2(stdout, 1); err != nil {
		return fmt.Errorf("dup2 stdout from %d: %w", stdout, err)
	}

	if err := syscall.Dup2(stderr, 2); err != nil {
		return fmt.Errorf("dup2 stderr from %d: %w", stderr, err)
	}

	return nil
}
