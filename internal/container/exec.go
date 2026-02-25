package container

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// ExecOpts holds the options for executing a command in an existing container.
type ExecOpts struct {
	// TODO: This is getting pretty big. Perhaps we want to restructure.
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

	procAttr := &syscall.ProcAttr{Sys: &syscall.SysProcAttr{}}

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
		defer func() {
			if err := ptySocket.Close(); err != nil {
				slog.Warn("failed to close pty socket", "container_pid", containerPID, "err", err)
			}
		}()

		pty, err := terminal.NewPty()
		if err != nil {
			return 0, fmt.Errorf("create pty pair: %w", err)
		}
		defer func() {
			if err := pty.Master.Close(); err != nil {
				slog.Warn("failed to close pty master", "container_pid", containerPID, "err", err)
			}
		}()

		if err := terminal.SendPty(ptySocket.SocketFd, pty); err != nil {
			return 0, fmt.Errorf("send pty: %w", err)
		}

		procAttr.Files = []uintptr{pty.Slave.Fd(), pty.Slave.Fd(), pty.Slave.Fd()}

		args = append(args, "--tty")
	} else {
		procAttr.Files = []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}
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
			slog.Debug("skip shared namespace", "container_path", containerNSPath, "host_path", hostNSPath)
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
			defer func() {
				if err := pidNSFile.Close(); err != nil {
					slog.Warn("failed to close PID namespace file", "pid_file", pidNSFile.Name(), "container_pid", containerPID, "err", err)
				}
			}()

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
			fmt.Sprintf("%s=%s", envJoinNS, strings.Join(joinNSParts, ",")),
		)
	}

	// Pass container PID so C constructor can chroot into container's root.
	procAttr.Env = append(
		procAttr.Env,
		fmt.Sprintf("%s=%d", envContainerPID, containerPID),
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

	// Check if executable exists in the container's filesystem before forking.
	// This allows returning an error synchronously (as required by containerd)
	// rather than deferring failure to later.
	containerRoot := fmt.Sprintf("/proc/%d/root", containerPID)
	if err := checkExecutableInContainer(containerRoot, opts.Args[0], opts.Env); err != nil {
		return 0, fmt.Errorf("check executable in container: %w", err)
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
		if err := os.WriteFile(opts.PIDFile, strconv.AppendInt(nil, int64(pid), 10), 0o644); err != nil {
			return 0, fmt.Errorf("write pid to file (%s): %w", opts.PIDFile, err)
		}
	}

	if !opts.Detach {
		var ws unix.WaitStatus

		slog.Debug("waiting for process", "pid", pid)

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

// checkExecutableInContainer verifies that the executable exists and is executable
// within the container's filesystem. containerRoot should be /proc/<pid>/root.
func checkExecutableInContainer(containerRoot, exe string, env []string) error {
	if strings.HasPrefix(exe, "/") {
		if err := checkExecutable(filepath.Join(containerRoot, exe)); err != nil {
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

		fullPath := filepath.Join(containerRoot, dir, exe)
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
