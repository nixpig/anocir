package container

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"syscall"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ExecOpts holds the options for executing a command in an existing container.
type ExecOpts struct {
	Cwd            string
	Args           []string
	UID            int
	GID            int
	PIDFile        string
	TTY            bool
	Detach         bool
	Env            []string
	AdditionalGIDs []int
	NoNewPrivs     bool
	Capabilities   []string
	ConsoleSocket  string
	ContainerID    string

	// TODO: Handle these options.
	IgnorePaused bool
	PreserveFDs  int
	ProcessLabel string
	AppArmor     string
	Cgroup       string
}

// ChildExecOpts holds the options for the forked process that executes a
// command in an existing container.
type ChildExecOpts struct {
	Cwd         string
	Args        []string
	ContainerID string

	Env          []string
	Capabilities *specs.LinuxCapabilities
	User         *specs.User
	NoNewPrivs   bool
	TTY          bool
}

// Namespaces need to be applied in a specific order. Don't change these.
var namespaces = []string{
	"user",
	"pid",
	"net",
	"ipc",
	"uts",
	"cgroup",
	"time",
	"mnt",
}

// Exec configures the execution of a command with the given opts in an
// existing container with the given containerPID. It forks a child process to
// handle the execution and returns the exit code.
func Exec(containerPID int, opts *ExecOpts) (int, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	additionalGIDs := make([]string, 0, len(opts.AdditionalGIDs))
	for _, g := range opts.AdditionalGIDs {
		additionalGIDs = append(additionalGIDs, fmt.Sprintf("%d", g))
	}

	procAttr := &syscall.ProcAttr{
		Sys: &syscall.SysProcAttr{},
	}

	args := []string{
		"childexec",
		"--cwd", opts.Cwd,
		"--uid", fmt.Sprintf("%d", opts.UID),
		"--gid", fmt.Sprintf("%d", opts.GID),
		"--container-id", opts.ContainerID,
	}

	if opts.ConsoleSocket != "" {
		ptySocket, err := terminal.NewPtySocket(opts.ConsoleSocket)
		if err != nil {
			return 0, fmt.Errorf("create pty socket: %w", err)
		}
		defer ptySocket.Close()

		pty, err := terminal.NewPty()
		if err != nil {
			return 0, fmt.Errorf("create pty pair: %w", err)
		}
		defer pty.Master.Close()

		if err := terminal.SendPty(ptySocket.SocketFd, pty); err != nil {
			return 0, fmt.Errorf("send pty: %w", err)
		}
		procAttr.Files = []uintptr{
			pty.Slave.Fd(),
			pty.Slave.Fd(),
			pty.Slave.Fd(),
		}

		args = append(args, "--tty")
	} else {
		procAttr.Files = []uintptr{
			os.Stdin.Fd(),
			os.Stdout.Fd(),
			os.Stderr.Fd(),
		}
	}

	if opts.NoNewPrivs {
		args = append(args, "--no-new-privs")
	}

	args = appendArgsSlice(args, "--additional-gids", additionalGIDs)
	args = appendArgsSlice(args, "--caps", opts.Capabilities)
	args = appendArgsSlice(args, "--envs", opts.Env)
	args = appendArgsSlice(args, "--args", opts.Args)

	for _, ns := range namespaces {
		if ns == "time" {
			// TODO: Apply this - gonna need to be time_for_children or skip, I think.
			continue
		}

		containerNSPath := fmt.Sprintf("/proc/%d/ns/%s", containerPID, ns)
		hostNSPath := fmt.Sprintf("/proc/1/ns/%s", ns)

		isSharedNS, err := sharedNamespace(containerNSPath, hostNSPath)
		if err != nil {
			return 0, fmt.Errorf("check if shared namespace: %w", err)
		}

		if isSharedNS {
			// If container shares host namespace then nothing to do.
			continue
		}

		if ns == "user" {
			procAttr.Sys.Credential = &syscall.Credential{Uid: 0, Gid: 0}
		}

		if ns == "mnt" {
			procAttr.Env = append(
				procAttr.Env,
				fmt.Sprintf("%s=%s", envMountNS, containerNSPath),
			)
		} else {
			if err := platform.SetNS(containerNSPath); err != nil {
				return 0, fmt.Errorf("join namespace: %w", err)
			}
		}
	}

	procAttr.Env = append(procAttr.Env, opts.Env...)

	// TODO: This is going to leak host environment into container. Probably not
	// what we want to do.
	procAttr.Env = append(procAttr.Env, os.Environ()...)

	_, err := exec.LookPath(opts.Args[0])
	if err != nil {
		return 0, fmt.Errorf("check binary exists: %w", err)
	}

	execArgs := append([]string{"/proc/self/exe"}, args...)

	slog.Debug(
		"child fork exec",
		"container_id", opts.ContainerID,
		"container_pid", containerPID,
		"argv0", execArgs[0],
		"argv", execArgs,
		"attr", procAttr,
	)

	pid, err := syscall.ForkExec(execArgs[0], execArgs, procAttr)
	if err != nil {
		return 0, fmt.Errorf("reexec child process: %w", err)
	}

	if opts.PIDFile != "" {
		if err := os.WriteFile(opts.PIDFile, fmt.Appendf(nil, "%d", pid), 0o644); err != nil {
			return 0, fmt.Errorf(
				"write pid to file (%s): %w",
				opts.PIDFile,
				err,
			)
		}
	}

	if !opts.Detach {
		var ws unix.WaitStatus
		if _, err := unix.Wait4(pid, &ws, 0, nil); err != nil {
			return 0, fmt.Errorf("wait for child process: %w", err)
		}

		if ws.Exited() {
			return ws.ExitStatus(), nil
		}

		if ws.Signaled() {
			sig := ws.Signal()
			return 128 + int(sig), nil
		}
	}

	return 0, nil
}

// ChildExec handles the execution of a command in an existing container with
// the given opts.
func ChildExec(opts *ChildExecOpts) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if opts.Capabilities != nil {
		if err := platform.DropBoundingCapabilities(opts.Capabilities); err != nil {
			return fmt.Errorf("drop bounding caps: %w", err)
		}
	}

	if opts.User != nil && opts.User.UID != 0 && opts.Capabilities != nil {
		if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0); err != nil {
			return fmt.Errorf("set KEEPCAPS: %w", err)
		}
	}

	if err := platform.SetUser(opts.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	if opts.Capabilities != nil {
		if err := platform.SetCapabilities(opts.Capabilities); err != nil {
			return fmt.Errorf("set capabilities: %w", err)
		}

		if opts.User.UID != 0 {
			if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 0, 0, 0, 0); err != nil {
				return fmt.Errorf("clear KEEPCAPS: %w", err)
			}
		}
	}

	if opts.NoNewPrivs {
		if err := platform.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	if opts.Cwd != "" {
		if err := os.Chdir(opts.Cwd); err != nil {
			return fmt.Errorf("change working directory: %w", err)
		}
	}

	if opts.TTY {
		if _, err := unix.Setsid(); err != nil {
			return fmt.Errorf("setsid: %w", err)
		}

		if err := unix.IoctlSetInt(0, unix.TIOCSCTTY, 0); err != nil {
			return fmt.Errorf("set ioctl: %w", err)
		}
	}

	bin, err := exec.LookPath(opts.Args[0])
	if err != nil {
		return fmt.Errorf("find path of binary: %w", err)
	}

	slog.Debug(
		"execute child process",
		"container_id", opts.ContainerID,
		"bin", bin,
		"args", opts.Args,
		"additional_envs", opts.Env,
	)

	envs := slices.Concat(os.Environ(), opts.Env)
	if err := unix.Exec(bin, opts.Args, envs); err != nil {
		return fmt.Errorf(
			"execve (argv0=%s, argv=%s, envv=%v): %w",
			bin, opts.Args, os.Environ(), err,
		)
	}

	panic("unreachable")
}

// sharedNamespace determines whether the host and container share a namespace
// by checking whether the containerNSPath and hostNSPath are the same file.
func sharedNamespace(containerNSPath, hostNSPath string) (bool, error) {
	containerNS, err := os.Stat(containerNSPath)
	if err != nil {
		return false, fmt.Errorf("stat container path: %w", err)
	}

	hostNS, err := os.Stat(hostNSPath)
	if err != nil {
		return false, fmt.Errorf("stat host path: %w", err)
	}

	if os.SameFile(containerNS, hostNS) {
		return true, nil
	}

	return false, nil
}

func appendArgsSlice(args []string, flag string, values []string) []string {
	for _, a := range values {
		args = append(args, flag, a)
	}

	return args
}
