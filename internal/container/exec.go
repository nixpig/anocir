package container

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
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
	Seccomp        *specs.LinuxSeccomp

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
	Seccomp      *specs.LinuxSeccomp
}

// Namespaces need to be applied in a specific order. Don't change these.
var namespaces = []specs.LinuxNamespaceType{
	specs.UserNamespace,
	specs.PIDNamespace,
	specs.NetworkNamespace,
	specs.IPCNamespace,
	specs.UTSNamespace,
	specs.CgroupNamespace,
	specs.TimeNamespace,
	specs.MountNamespace,
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

	var joinNSParts []string

	for _, ns := range namespaces {
		nsName, ok := platform.NamespaceEnvs[ns]
		if !ok {
			return 0, fmt.Errorf("unknown namespace: %s", ns)
		}

		containerNSPath := fmt.Sprintf("/proc/%d/ns/%s", containerPID, nsName)
		hostNSPath := fmt.Sprintf("/proc/1/ns/%s", nsName)

		isSharedNS, err := sharedNamespace(containerNSPath, hostNSPath)
		if err != nil {
			return 0, fmt.Errorf("check if shared namespace: %w", err)
		}

		if isSharedNS {
			// If container shares host namespace then nothing to do.
			continue
		}

		if ns == specs.TimeNamespace {
			// TODO: Apply this - gonna need to be time_for_children or skip, I think.
			continue
		}

		if ns == specs.PIDNamespace {
			// For PID namespace, we call setns() here in the parent so the
			// subsequent ForkExec will create a child that is actually in the
			// container's PID namespace.
			pidNSFile, err := os.Open(containerNSPath)
			if err != nil {
				return 0, fmt.Errorf("open pid namespace: %w", err)
			}
			defer pidNSFile.Close()

			if err := unix.Setns(int(pidNSFile.Fd()), unix.CLONE_NEWPID); err != nil {
				return 0, fmt.Errorf("setns to pid namespace: %w", err)
			}
			continue
		}

		if ns == specs.UserNamespace {
			procAttr.Sys.Credential = &syscall.Credential{Uid: 0, Gid: 0}
			continue
		}

		// TODO: This is exactly the same as in container.go; maybe factor out.
		f, err := os.Open(containerNSPath)
		if err != nil {
			return 0, fmt.Errorf("open ns path: %w", err)
		}

		joinNSParts = append(
			joinNSParts,
			fmt.Sprintf("%s:%s", platform.NamespaceEnvs[ns], containerNSPath),
		)
		f.Close()
	}

	procAttr.Env = append(procAttr.Env, opts.Env...)

	if len(joinNSParts) > 0 {
		procAttr.Env = append(
			procAttr.Env,
			fmt.Sprintf("_ANOCIR_JOIN_NS=%s", strings.Join(joinNSParts, ",")),
		)
	}

	// Pass container PID so C constructor can chroot into container's root.
	procAttr.Env = append(
		procAttr.Env,
		fmt.Sprintf("_ANOCIR_CONTAINER_PID=%d", containerPID),
	)

	// Pass seccomp profile by writing to container's filesystem,
	// /proc/<pid>/root/tmp appears as /tmp inside the container.
	//
	// TODO: This is pretty hacky and should probably find a cleaner way of
	// passing the seccomp profile to the exec.
	if opts.Seccomp != nil {
		seccompData, err := json.Marshal(opts.Seccomp)
		if err != nil {
			return 0, fmt.Errorf("serialise seccomp profile: %w", err)
		}

		containerTmp := fmt.Sprintf("/proc/%d/root/tmp", containerPID)
		if err := os.MkdirAll(containerTmp, 0o755); err != nil {
			return 0, fmt.Errorf("create container tmp dir: %w", err)
		}

		seccompPath := filepath.Join(containerTmp, fmt.Sprintf("seccomp-%d.json", os.Getpid()))
		if err := os.WriteFile(seccompPath, seccompData, 0o644); err != nil {
			return 0, fmt.Errorf("write seccomp profile: %w", err)
		}

		// The path inside the container is /tmp/seccomp-<pid>.json
		containerSeccompPath := fmt.Sprintf("/tmp/seccomp-%d.json", os.Getpid())
		args = append(args, "--seccomp-file", containerSeccompPath)
	}

	// Check if binary exists in the container's filesystem before forking.
	// This allows returning an error synchronously (as required by containerd)
	// rather than deferring failure to later.
	containerRoot := fmt.Sprintf("/proc/%d/root", containerPID)
	if err := checkBinaryInContainer(
		containerRoot,
		opts.Args[0],
		opts.Env,
	); err != nil {
		return 0, fmt.Errorf("check binary in container: %w", err)
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

	// Namespace joining and chroot to container root is handled by the
	// C constructor (nssetup) which runs before Go starts.

	// TODO: We have a lot of overlap between exec and pre-pivot. Review whether
	// some of this can be factored out to reduce unnecessary duplication.

	// When NoNewPrivileges is false, we load seccomp BEFORE dropping
	// capabilities because seccomp filter loading is a privileged operation that
	// requires CAP_SYS_ADMIN when NO_NEW_PRIVS is not set.
	//
	// See: https://man7.org/linux/man-pages/man2/seccomp.2.html
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

	envs := slices.Concat(os.Environ(), opts.Env)
	for _, env := range envs {
		e := strings.SplitN(env, "=", 2)
		if len(e) == 2 {
			os.Setenv(e[0], e[1])
		} else {
			slog.Debug("invalid environment var", "env", env)
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

// checkBinaryInContainer verifies that the binary exists and is executable
// within the container's filesystem. containerRoot should be /proc/<pid>/root.
func checkBinaryInContainer(containerRoot, binary string, env []string) error {
	if strings.HasPrefix(binary, "/") {
		fullPath := filepath.Join(containerRoot, binary)
		if err := checkExecutable(fullPath); err != nil {
			return fmt.Errorf("executable file not found in $PATH: %w", err)
		}
		return nil
	}

	// Get PATH from environment.
	pathEnv := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	for _, e := range env {
		if after, ok := strings.CutPrefix(e, "PATH="); ok {
			pathEnv = after
			break
		}
	}

	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}

		fullPath := filepath.Join(containerRoot, dir, binary)
		if err := checkExecutable(fullPath); err == nil {
			return nil
		}
	}

	return fmt.Errorf("executable file not found in $PATH")
}

// checkExecutable verifies that a file exists and is executable.
func checkExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	// Check if any execute bit is set.
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable", path)
	}

	return nil
}
