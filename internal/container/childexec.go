package container

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ChildExecOpts holds the options for the forked process that executes a
// command in an existing container.
type ChildExecOpts struct {
	Cwd          string
	Args         []string
	ContainerID  string
	Env          []string
	Capabilities *specs.LinuxCapabilities
	User         *specs.User
	NoNewPrivs   bool
	TTY          bool
	Seccomp      *specs.LinuxSeccomp
}

// ChildExec handles the execution of a command in an existing container with
// the given opts.
func ChildExec(opts *ChildExecOpts) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Namespace joining and chroot to container root is handled by the
	// C constructor (nssetup) which runs before Go starts.

	// TODO: We have a lot of overlap between exec and pre-pivot. Review whether
	// some of this can be factored out to reduce unnecessary duplication.

	// When NoNewPrivileges is false, we load seccomp BEFORE dropping
	// capabilities because seccomp filter loading is a privileged operation that
	// requires CAP_SYS_ADMIN when NO_NEW_PRIVS is not set.
	if opts.Seccomp != nil && !opts.NoNewPrivs {
		if err := platform.LoadSeccompFilter(opts.Seccomp); err != nil {
			return fmt.Errorf("load seccomp filter (privileged): %w", err)
		}
	}

	if opts.Capabilities != nil {
		if err := platform.DropBoundingCapabilities(opts.Capabilities); err != nil {
			return fmt.Errorf("drop bounding caps: %w", err)
		}
	}

	if opts.User != nil && opts.User.UID != 0 && opts.Capabilities != nil {
		if err := platform.SetKeepCaps(1); err != nil {
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

		if opts.User != nil && opts.User.UID != 0 {
			if err := platform.SetKeepCaps(0); err != nil {
				return fmt.Errorf("clear KEEPCAPS: %w", err)
			}
		}
	}

	if opts.NoNewPrivs {
		if err := platform.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	// When NoNewPrivileges is true, we load seccomp AFTER setting NO_NEW_PRIVS,
	// as close to execve as possible to minimize the syscall surface. The
	// NO_NEW_PRIVS bit allows unprivileged seccomp filter loading.
	if opts.Seccomp != nil && opts.NoNewPrivs {
		if err := platform.LoadSeccompFilter(opts.Seccomp); err != nil {
			return fmt.Errorf("load seccomp filter: %w", err)
		}
	}

	if opts.Cwd != "" {
		if err := unix.Chdir(opts.Cwd); err != nil {
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

	envs := slices.Concat(unix.Environ(), opts.Env)
	for _, env := range envs {
		e := strings.SplitN(env, "=", 2)
		if len(e) == 2 {
			unix.Setenv(e[0], e[1])
		} else {
			slog.Debug("invalid environment var", "env", env)
		}
	}

	exe, err := exec.LookPath(opts.Args[0])
	if err != nil {
		return fmt.Errorf("find path of binary: %w", err)
	}

	slog.Debug(
		"execute child process",
		"container_id", opts.ContainerID,
		"exe", exe,
		"args", opts.Args,
		"additional_envs", opts.Env,
	)

	if err := unix.Exec(exe, opts.Args, envs); err != nil {
		return fmt.Errorf(
			"execve (argv0=%s, argv=%s, envv=%v): %w",
			exe, opts.Args, envs, err,
		)
	}

	panic("unreachable")
}
