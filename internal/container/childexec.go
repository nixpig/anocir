package container

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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
	Cwd             string
	Args            []string
	ContainerID     string
	Env             []string
	Capabilities    *specs.LinuxCapabilities
	User            *specs.User
	NoNewPrivs      bool
	TTY             bool
	Seccomp         *specs.LinuxSeccomp
	AppArmorProfile string
	ProcessLabel    string
	Cgroup          string
}

// ChildExec handles the execution of a command in an existing container with
// the given opts.
func ChildExec(opts *ChildExecOpts) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Note: namespace joining and chroot to container root is handled by the
	// C constructor (nssetup) which runs before Go starts.

	if err := platform.ApplyProcessSecurity(&platform.ProcessSecurity{
		User:            opts.User,
		Capabilities:    opts.Capabilities,
		Seccomp:         opts.Seccomp,
		NoNewPrivs:      opts.NoNewPrivs,
		AppArmorProfile: opts.AppArmorProfile,
		ProcessLabel:    opts.ProcessLabel,
	}); err != nil {
		return fmt.Errorf("apply process security: %w", err)
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

	if opts.Cgroup != "" {
		contents, err := os.ReadFile("/proc/self/cgroup")
		if err != nil {
			return fmt.Errorf("open cgroup path: %w", err)
		}

		var cntrCgroupPath string
		for s := range strings.Lines(string(contents)) {
			if p, ok := strings.CutPrefix(s, "0::"); ok {
				cntrCgroupPath = filepath.Join(
					"/sys/fs/cgroup",
					strings.TrimSpace(p),
					opts.Cgroup,
					"cgroup.procs",
				)

				break
			}
		}

		if cntrCgroupPath == "" {
			return errors.New("no cgroup v2 entry found in /sys/fs/cgroup")
		}

		if err := os.WriteFile(cntrCgroupPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
			return fmt.Errorf("write pid to cgroup: %w", err)
		}
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
