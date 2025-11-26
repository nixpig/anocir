package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"

	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// readyMsg is the message sent over the init socketpair when the container
// is created and ready to receive commands.
const readyMsg = "ready"

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

	if err := ipc.SendMessage(conn, readyMsg); err != nil {
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

func (c *Container) setupPrePivot() error {
	if err := platform.MountRootfs(c.rootFS()); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	if err := platform.MountProc(c.rootFS()); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	c.spec.Mounts = append(c.spec.Mounts, specs.Mount{
		Destination: "/dev/pts",
		Type:        "devpts",
		Source:      "devpts",
		Options: []string{
			"nosuid",
			"noexec",
			"newinstance",
			"ptmxmode=0666",
			"mode=0620",
			"gid=5",
		},
	})

	if err := platform.MountSpecMounts(c.spec.Mounts, c.rootFS()); err != nil {
		return fmt.Errorf("mount spec mounts: %w", err)
	}

	if err := platform.MountDefaultDevices(c.rootFS()); err != nil {
		return fmt.Errorf("mount default devices: %w", err)
	}

	if err := platform.CreateDeviceNodes(c.spec.Linux.Devices, c.rootFS()); err != nil {
		return fmt.Errorf("mount devices from spec: %w", err)
	}

	if err := platform.CreateDefaultSymlinks(c.rootFS()); err != nil {
		return fmt.Errorf("create default symlinks: %w", err)
	}

	return nil
}

func (c *Container) setupPostPivot() error {
	if c.spec.Linux.Sysctl != nil {
		if err := platform.SetSysctl(c.spec.Linux.Sysctl); err != nil {
			return fmt.Errorf("set sysctl: %w", err)
		}
	}

	if err := platform.MountMaskedPaths(c.spec.Linux.MaskedPaths); err != nil {
		return fmt.Errorf("mount masked paths: %w", err)
	}

	if err := platform.MountReadonlyPaths(c.spec.Linux.ReadonlyPaths); err != nil {
		return fmt.Errorf("mount readonly paths: %w", err)
	}

	if err := platform.SetRootfsMountPropagation(
		c.spec.Linux.RootfsPropagation,
	); err != nil {
		return fmt.Errorf("set rootfs mount propagation: %w", err)
	}

	if c.spec.Root.Readonly {
		if err := platform.MountRootReadonly(); err != nil {
			return fmt.Errorf("remount root as readonly: %w", err)
		}
	}

	hasUTSNamespace := slices.ContainsFunc(
		c.spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	)

	if hasUTSNamespace {
		if err := unix.Sethostname([]byte(c.spec.Hostname)); err != nil {
			return fmt.Errorf("set hostname: %w", err)
		}

		if err := unix.Setdomainname([]byte(c.spec.Domainname)); err != nil {
			return fmt.Errorf("set domainname: %w", err)
		}
	}

	if err := platform.SetRlimits(c.spec.Process.Rlimits); err != nil {
		return fmt.Errorf("set rlimits: %w", err)
	}

	if c.spec.Process.Capabilities != nil {
		if err := platform.SetCapabilities(c.spec.Process.Capabilities); err != nil {
			return fmt.Errorf("set capabilities: %w", err)
		}
	}

	if c.spec.Process.NoNewPrivileges {
		if err := platform.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	if c.spec.Process.Scheduler != nil {
		schedAttr, err := platform.NewSchedAttr(c.spec.Process.Scheduler)
		if err != nil {
			return fmt.Errorf("new sched attr: %w", err)
		}

		if err := platform.SchedSetAttr(schedAttr); err != nil {
			return fmt.Errorf("set sched attr: %w", err)
		}
	}

	if c.spec.Process.IOPriority != nil {
		ioprio, err := platform.IOPrioToInt(c.spec.Process.IOPriority)
		if err != nil {
			return fmt.Errorf("convert ioprio to int: %w", err)
		}

		if err := platform.IOPrioSet(ioprio); err != nil {
			return fmt.Errorf("set ioprio: %w", err)
		}
	}

	if err := platform.SetUser(&c.spec.Process.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	return nil
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
	if msg != startMsg {
		return fmt.Errorf("expecting '%s' but received '%s'", startMsg, msg)
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
