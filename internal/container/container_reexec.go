package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"golang.org/x/sys/unix"
)

// ErrMissingProcess is returned when the provided spec is missing a process.
var ErrMissingProcess = errors.New("process is required")

// Reexec is the entry point for the containerised process. It is responsible
// for setting up the Container environment, including namespaces, mounts,
// and security settings, before executing the user-specified process.
func (c *Container) Reexec() error {
	// Subsequent syscalls need to happen in a single-threaded context.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := c.connectConsole(); err != nil {
		return err
	}

	if err := c.setupPrePivot(); err != nil {
		return err
	}

	if err := c.mountConsole(); err != nil {
		return err
	}

	initSockFD := os.Getenv(envInitSockFD)
	if initSockFD == "" {
		return errors.New("missing init sock fd")
	}

	initSockFDVal, err := strconv.Atoi(initSockFD)
	if err != nil {
		return errors.New("invalid init sock fd")
	}

	conn, err := ipc.FDToConn(initSockFDVal)
	if err != nil {
		return err
	}

	if err := ipc.SendMessage(conn, ipc.ReadyMsg); err != nil {
		return fmt.Errorf("failed to send ready message: %w", err)
	}

	conn.Close()

	if err := c.waitStart(); err != nil {
		return err
	}

	if err := c.pivotRoot(); err != nil {
		return err
	}

	if err := c.setupPostPivot(); err != nil {
		return err
	}

	if err := c.execHooks(LifecycleStartContainer); err != nil {
		return fmt.Errorf("exec startcontainer hooks: %w", err)
	}

	if err := c.execUserProcess(); err != nil {
		return err
	}

	panic("if you got here then something wrong that is not recoverable")
}

func (c *Container) execUserProcess() error {
	if err := os.Chdir(c.spec.Process.Cwd); err != nil {
		return fmt.Errorf("set working directory: %w", err)
	}

	bin, err := exec.LookPath(c.spec.Process.Args[0])
	if err != nil {
		return fmt.Errorf("find path of user process binary: %w", err)
	}

	if err := unix.Exec(bin, c.spec.Process.Args, os.Environ()); err != nil {
		return fmt.Errorf(
			"execve (argv0=%s, argv=%s, envv=%v): %w",
			bin, c.spec.Process.Args, os.Environ(), err,
		)
	}

	return nil
}

// waitStart listens on the container socket for the start message.
func (c *Container) waitStart() error {
	containerSock := ipc.NewSocket(c.containerSock)
	listener, err := containerSock.Listen()
	if err != nil {
		return fmt.Errorf("listen on container sock: %w", err)
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept on container sock: %w", err)
	}
	defer conn.Close()

	msg, err := ipc.ReceiveMessage(conn)
	if err != nil {
		return fmt.Errorf("read from container sock: %w", err)
	}
	if msg != ipc.StartMsg {
		return fmt.Errorf("expecting '%s' but received '%s'", ipc.StartMsg, msg)
	}

	return nil
}

func (c *Container) mountConsole() error {
	if c.pty == nil || !c.spec.Process.Terminal {
		return nil
	}

	target := filepath.Join(c.rootFS(), "dev/console")

	if err := c.pty.MountSlave(target); err != nil {
		return err
	}

	return nil
}

func (c *Container) connectConsole() error {
	if c.ConsoleSocketFD == 0 {
		return nil
	}

	pty, err := terminal.NewPty()
	if err != nil {
		return fmt.Errorf("new pty: %w", err)
	}

	if c.spec.Process.ConsoleSize != nil {
		if err := platform.SetWinSize(
			pty.Slave.Fd(),
			c.spec.Process.ConsoleSize.Width,
			c.spec.Process.ConsoleSize.Height,
		); err != nil {
			return fmt.Errorf("set console size: %w", err)
		}
	}

	if err := terminal.SendPty(c.ConsoleSocketFD, pty); err != nil {
		return fmt.Errorf("connect pty and socket: %w", err)
	}

	unix.Close(c.ConsoleSocketFD)

	if err := pty.Connect(); err != nil {
		return fmt.Errorf("connect pty: %w", err)
	}

	c.pty = pty

	return nil
}

func (c *Container) pivotRoot() error {
	if c.spec.Process == nil {
		return ErrMissingProcess
	}

	if err := platform.PivotRoot(c.rootFS()); err != nil {
		return err
	}
	return nil
}
