package container

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

type ExecOpts struct {
	ContainerPID   int
	Rootfs         string
	Cwd            string
	Args           []string
	ConsoleSocket  string
	UID            int
	GID            int
	PIDFile        string
	TTY            bool
	Detach         bool
	IgnorePaused   bool
	PreserveFDs    int
	Env            []string
	AdditionalGIDs []int
	Process        string
	ProcessLabel   string
	AppArmor       string
	NoNewPrivs     bool
	Capabilities   []string
	Cgroup         string
}

type ChildExecOpts struct {
	Cwd  string
	Args []string

	Capabilities *specs.LinuxCapabilities
	User         *specs.User
	NoNewPrivs   bool
}

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

func Exec(opts *ExecOpts) (int, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	additionalGIDs := make([]string, 0, len(opts.AdditionalGIDs))
	for _, g := range opts.AdditionalGIDs {
		additionalGIDs = append(additionalGIDs, fmt.Sprintf("%d", g))
	}

	procAttr := &syscall.ProcAttr{
		Sys: &syscall.SysProcAttr{},
	}

	procAttr.Files = []uintptr{
		os.Stdin.Fd(),
		os.Stdout.Fd(),
		os.Stderr.Fd(),
	}

	args := []string{
		"childexec",
		"--cwd", opts.Cwd,
		"--uid", fmt.Sprintf("%d", opts.UID),
		"--gid", fmt.Sprintf("%d", opts.GID),
	}

	if len(additionalGIDs) > 0 {
		args = append(
			args,
			"--additional-gids", strings.Join(additionalGIDs, ","),
		)
	}

	if len(opts.Capabilities) > 0 {
		args = append(
			args,
			"--caps", strings.Join(opts.Capabilities, ","),
		)
	}

	if len(opts.Env) > 0 {
		args = append(
			args,
			"--envs", strings.Join(opts.Env, ","),
		)
	}

	if len(opts.Args) > 0 {
		args = append(
			args,
			"--args", strings.Join(opts.Args, ","),
		)
	}

	if opts.NoNewPrivs {
		args = append(args, "--no-new-privs")
	}

	for _, ns := range namespaces {
		if ns == "time" {
			// TODO: Apply this - gonna need to be time_for_children or skip, I think.
			continue
		}

		containerNSPath := fmt.Sprintf("/proc/%d/ns/%s", opts.ContainerPID, ns)
		hostNSPath := fmt.Sprintf("/proc/1/ns/%s", ns)

		isSharedNS, err := sharedNamespace(containerNSPath, hostNSPath)
		if err != nil {
			return 255, fmt.Errorf("check if shared namespace: %w", err)
		}

		if isSharedNS {
			// If container shares host namespace then nothing to do.
			continue
		}

		if ns == "user" {
			procAttr.Sys.Credential = &syscall.Credential{Uid: 0, Gid: 0}
		}

		if ns == "mnt" {
			gonsEnv := fmt.Sprintf("gons_mnt=%s", containerNSPath)
			procAttr.Env = append(procAttr.Env, gonsEnv)
		} else {
			if err := platform.SetNS(containerNSPath); err != nil {
				return 255, fmt.Errorf("join namespace: %w", err)
			}
		}
	}

	procAttr.Env = append(procAttr.Env, opts.Env...)
	procAttr.Env = append(procAttr.Env, os.Environ()...)

	execArgs := append([]string{"/proc/self/exe"}, args...)

	pid, err := syscall.ForkExec(execArgs[0], execArgs, procAttr)
	if err != nil {
		return 255, fmt.Errorf("reexec child process: %w", err)
	}

	if !opts.Detach {
		var ws unix.WaitStatus
		if _, err := unix.Wait4(pid, &ws, 0, nil); err != nil {
			return 255, fmt.Errorf("wait for child process: %w", err)
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

	bin, err := exec.LookPath(opts.Args[0])
	if err != nil {
		return fmt.Errorf("find path of binary: %w", err)
	}

	if err := unix.Exec(bin, opts.Args, os.Environ()); err != nil {
		return fmt.Errorf(
			"execve (argv0=%s, argv=%s, envv=%v): %w",
			bin, opts.Args, os.Environ(), err,
		)
	}

	panic("unreachable")
}

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
