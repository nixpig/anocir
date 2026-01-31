// Package container provides functionality for creating, running, and managing
// OCI-compliant containers.
package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

const (
	// containerSockFilename is the filename of the socket used by the runtime to
	// send messages to the container.
	containerSockFilename = "c.sock"

	// lockFilename is the filename of the lockfile used to synchronise access
	// to container operations.
	lockFilename = "c.lock"

	// envInitSockFD is the name of the environment variable used to pass the
	// init socket file descriptor to the reexec'd process.
	envInitSockFD = "_ANOCIR_INIT_SOCK_FD"

	// envContainerSockFD is the name of the environment variable used to pass
	// the container socket file descriptor to the reexec'd process.
	envContainerSockFD = "_ANOCIR_CONTAINER_SOCK_FD"

	// envJoinNS is the name of the environment variable used to pass the
	// namespaces to join to the reexec'd process.
	envJoinNS = "_ANOCIR_JOIN_NS"
)

// ErrOperationInProgress is returned when the container is locked by another
// operation.
var ErrOperationInProgress = errors.New("operation already in progress")

// Container represents an OCI container, including its state, specification,
// and other runtime details.
type Container struct {
	State           *specs.State
	ConsoleSocket   string
	ConsoleSocketFD int
	RootDir         string
	LogFormat       string
	LogFile         string

	spec          *specs.Spec
	pty           *terminal.Pty
	pidFile       string
	containerSock string
	lockFile      *os.File
	debug         bool
}

// Opts holds the options for creating a new Container.
type Opts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
	RootDir       string
	LogFile       string
	Debug         bool
	LogFormat     string
}

// New constructs a Container based on the provided opts. The container will be
// in the 'creating' state.
func New(opts *Opts) (*Container, error) {
	if opts.Spec.Linux == nil {
		return nil, fmt.Errorf("spec must have Linux configuration")
	}

	state := &specs.State{
		Version:     specs.Version,
		ID:          opts.ID,
		Bundle:      opts.Bundle,
		Annotations: opts.Spec.Annotations,
		Status:      specs.StateCreating,
	}

	return &Container{
		State:         state,
		spec:          opts.Spec,
		ConsoleSocket: opts.ConsoleSocket,
		pidFile:       opts.PIDFile,
		debug:         opts.Debug,
		LogFormat:     opts.LogFormat,

		RootDir:       opts.RootDir,
		LogFile:       opts.LogFile,
		containerSock: containerSockPath(opts.Bundle),
	}, nil
}

func (c *Container) save() error {
	containerDir := filepath.Join(c.RootDir, c.State.ID)

	slog.Debug(
		"save state",
		"container_id", c.State.ID,
		"container_dir", containerDir,
		"state_filepath", c.stateFilepath(),
	)

	if len(c.spec.Linux.UIDMappings) > 0 && len(c.spec.Linux.GIDMappings) > 0 {
		if err := os.Chown(
			containerDir,
			int(c.spec.Linux.UIDMappings[0].HostID),
			int(c.spec.Linux.GIDMappings[0].HostID),
		); err != nil {
			return fmt.Errorf("chown container directory: %w", err)
		}
	}

	state, err := json.Marshal(c.State)
	if err != nil {
		return fmt.Errorf("serialise container state: %w", err)
	}

	if err := platform.AtomicWriteFile(
		c.stateFilepath(),
		state,
		0o644,
	); err != nil {
		return fmt.Errorf("write container state: %w", err)
	}

	if c.pidFile != "" && c.State.Pid > 0 {
		slog.Debug(
			"write pid file",
			"pid_file", c.pidFile,
			"pid", c.State.Pid,
		)

		if err := platform.AtomicWriteFile(
			c.pidFile,
			[]byte(strconv.Itoa(c.State.Pid)),
			0o644,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", c.pidFile, err)
		}
	}

	return nil
}

// Save persists the container state to disk. It creates the required directory
// hierarchy and sets the needed permissions.
func (c *Container) Save() error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	return c.save()
}

// Lock acquires an exclusive lock on the container.
func (c *Container) Lock() error {
	lockPath := filepath.Join(c.RootDir, c.State.ID, lockFilename)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()

		if err == unix.EWOULDBLOCK {
			return ErrOperationInProgress
		}

		return fmt.Errorf("acquire file lock: %w", err)
	}

	c.lockFile = f
	return nil
}

// Unlock releases the lock on the container.
func (c *Container) Unlock() error {
	if c.lockFile == nil {
		return nil
	}

	defer c.lockFile.Close()
	return unix.Flock(int(c.lockFile.Fd()), unix.LOCK_UN)
}

// Delete removes the container from the system. If force is true then it will
// delete the container, regardless of the its state.
func (c *Container) Delete(force bool) error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	slog.Debug("delete container", "container_id", c.State.ID, "force", force)

	if c.State.Pid != 0 {
		if err := unix.Kill(c.State.Pid, 0); err != nil {
			slog.Debug("process already dead", "pid", c.State.Pid)
			// Process is dead, update state
			c.State.Status = specs.StateStopped
		}
	}

	if !force && !c.canBeDeleted() {
		return fmt.Errorf(
			"container cannot be deleted in current state (%s) try using '--force'",
			c.State.Status,
		)
	}

	if err := platform.DeleteCgroup(c.spec.Linux.CgroupsPath, c.State.ID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to delete cgroup: %s", err.Error())
	}

	// TODO: Review whether need to remove pidfile.

	if err := os.RemoveAll(
		filepath.Join(c.RootDir, c.State.ID),
	); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	// Best effort.
	os.RemoveAll(filepath.Dir(c.containerSock))

	slog.Debug("execute poststop hooks", "container_id", c.State.ID)
	if err := c.execHooks(LifecyclePoststop); err != nil {
		fmt.Fprintf(
			os.Stdout,
			"Warning: failed to exec poststop hooks: %s\n",
			err,
		)
	}

	return nil
}

// GetState returns the state of the container. In the case the container
// process is no longer running, it updates the container state to be 'stopped'
// before returning.
func (c *Container) GetState() (*specs.State, error) {
	if err := c.Lock(); err != nil {
		return nil, fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return nil, fmt.Errorf("reload container state: %w", err)
	}

	if c.State.Pid != 0 {
		if err := unix.Kill(c.State.Pid, 0); err != nil {
			c.State.Status = specs.StateStopped
			if err := c.save(); err != nil {
				return nil, fmt.Errorf("save stopped state: %w", err)
			}
		}
	}

	return c.State, nil
}

func (c *Container) GetSpec() *specs.Spec {
	return c.spec
}

// Start begins the execution of the container by sending the start message to
// the runtime process.
func (c *Container) Start() error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	slog.Debug("start container", "container_id", c.State.ID)

	if !c.canBeStarted() {
		return fmt.Errorf(
			"container cannot be started in current state (%s)",
			c.State.Status,
		)
	}

	slog.Debug("execute prestart hooks", "container_id", c.State.ID)
	if err := c.execHooks(LifecyclePrestart); err != nil {
		return fmt.Errorf("execute prestart hooks: %w", err)
	}

	containerSock := ipc.NewSocket(c.containerSock)

	conn, err := containerSock.Dial()
	if err != nil {
		return fmt.Errorf("dial container sock: %w", err)
	}
	defer conn.Close()

	slog.Debug("send start message", "container_id", c.State.ID)
	if err := ipc.SendMessage(conn, ipc.MsgStart); err != nil {
		return fmt.Errorf(
			"write MsgStart ('%b') to container sock: %w",
			ipc.MsgStart,
			err,
		)
	}

	if c.spec.Process == nil {
		c.State.Status = specs.StateStopped
		if err := c.save(); err != nil {
			return fmt.Errorf("save state stopped: %w", err)
		}
		// Nothing to do; silent return.
		return nil
	}

	c.State.Status = specs.StateRunning
	if err := c.save(); err != nil {
		return fmt.Errorf("save state running: %w", err)
	}

	slog.Debug("execute poststart hooks", "container_id", c.State.ID)
	if err := c.execHooks(LifecyclePoststart); err != nil {
		return fmt.Errorf("exec poststart hooks: %w", err)
	}

	return nil
}

// Kill sends the given sig to the container process. If killAll is true then
// sig is sent to all processes in the container cgroup.
func (c *Container) Kill(sig string, killAll bool) error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	slog.Debug(
		"kill container",
		"container_id", c.State.ID,
		"signal", sig,
		"kill_all", killAll,
	)

	unixSig, err := platform.ParseSignal(sig)
	if err != nil {
		return fmt.Errorf("parse signal: %w", err)
	}

	if killAll {
		pids, err := platform.GetCgroupProcesses(
			c.spec.Linux.CgroupsPath,
			c.State.ID,
		)
		if err != nil {
			// cgroup may already be dead
			return nil
		}

		for _, pid := range pids {
			slog.Debug(
				"send kill signal",
				"container_id", c.State.ID,
				"cgroups_path", c.spec.Linux.CgroupsPath,
				"pid", pid,
				"signal", unixSig,
			)

			if err := platform.SendSignal(pid, unixSig); err != nil &&
				!errors.Is(err, unix.ESRCH) {
				return fmt.Errorf(
					"send killall signal '%s' to process '%d': %w",
					sig, pid, err,
				)
			}
		}
	} else {
		if !c.canBeKilled() {
			// The contents of this error message are important for passing containerd
			// integration test which does a string comparison.
			// https://github.com/containerd/containerd/blob/2d8e4b6d164e7566012fb6080fcc63dcde362628/cmd/containerd-shim-runc-v2/process/utils.go#L115C1-L128C2
			return fmt.Errorf("container not running (%s)", c.State.Status)
		}

		slog.Debug(
			"send kill signal",
			"container_id", c.State.ID,
			"pid", c.State.Pid,
			"signal", sig,
		)

		if err := platform.SendSignal(c.State.Pid, unixSig); err != nil {
			if errors.Is(err, unix.ESRCH) {
				return fmt.Errorf("container not running: %w", err)
			}

			return fmt.Errorf("send signal '%s' to process '%d': %w", sig, c.State.Pid, err)
		}
	}

	return nil
}

// Init prepares the container for execution. It executes hooks, sets up the
// terminal if necessary, and re-execs the runtime binary to containerise the
// process.
func (c *Container) Init() error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.save(); err != nil {
		return fmt.Errorf("save initial container state: %w", err)
	}

	slog.Debug(
		"init container",
		"container_id", c.State.ID,
		"bundle", c.State.Bundle,
	)

	args := []string{
		"reexec",
		"--root", c.RootDir,
		"--log-format", c.LogFormat,
		"--log", c.LogFile,
	}

	if c.useTerminal() {
		ptySocket, err := terminal.NewPtySocket(c.ConsoleSocket)
		if err != nil {
			return err
		}

		c.ConsoleSocketFD = ptySocket.SocketFd

		args = append(
			args,
			"--console-socket-fd",
			strconv.Itoa(c.ConsoleSocketFD),
		)
	}

	if c.debug {
		args = append(args, "--debug")
	}

	args = append(args, c.State.ID)

	cmd := exec.Command("/proc/self/exe", args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	initSockParent, initSockChild, err := ipc.NewSocketPair()
	if err != nil {
		return err
	}
	defer initSockParent.Close()
	defer initSockChild.Close()

	os.RemoveAll(filepath.Dir(c.containerSock))
	if err := os.MkdirAll(filepath.Dir(c.containerSock), 0o755); err != nil {
		return fmt.Errorf("create container socket directory: %w", err)
	}

	containerSock := ipc.NewSocket(c.containerSock)
	listener, err := containerSock.Listen()
	if err != nil {
		return fmt.Errorf("listen on container sock: %w", err)
	}

	unixListener, ok := listener.(*net.UnixListener)
	if !ok {
		return fmt.Errorf("expected UnixListener, got %T", listener)
	}

	listenerFile, err := unixListener.File()
	if err != nil {
		return fmt.Errorf("get listener file: %w", err)
	}
	defer listenerFile.Close()

	cmd.ExtraFiles = []*os.File{initSockChild, listenerFile}

	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf(
			"%s=%d",
			envInitSockFD,
			slices.Index(cmd.ExtraFiles, initSockChild)+3,
		),
		fmt.Sprintf(
			"%s=%d",
			envContainerSockFD,
			slices.Index(cmd.ExtraFiles, listenerFile)+3,
		),
	)

	if c.spec.Process != nil && c.spec.Process.OOMScoreAdj != nil {
		if err := platform.AdjustOOMScore(
			*c.spec.Process.OOMScoreAdj,
		); err != nil {
			return fmt.Errorf("adjust oom score: %w", err)
		}
	}

	// Subsequent syscalls need to happen in a single-threaded context.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := c.configureNamespaces(cmd); err != nil {
		return fmt.Errorf("configure namespaces: %w", err)
	}

	if c.spec.Process != nil && c.spec.Process.Env != nil {
		cmd.Env = append(cmd.Env, c.spec.Process.Env...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reexec container process: %w", err)
	}

	slog.Debug(
		"container process forked",
		"container_id", c.State.ID,
		"pid", cmd.Process.Pid,
	)

	c.State.Pid = cmd.Process.Pid

	slog.Debug(
		"create cgroup",
		"container_id", c.State.ID,
		"pid", c.State.Pid,
		"cgroups_path", c.spec.Linux.CgroupsPath,
	)
	if err := platform.CreateCgroup(
		c.spec.Linux.CgroupsPath,
		c.State.ID,
		c.State.Pid,
		c.spec.Linux.Resources,
	); err != nil {
		return fmt.Errorf("create cgroups: %w", err)
	}

	conn, err := net.FileConn(initSockParent)
	if err != nil {
		return fmt.Errorf("accept on init sock parent: %w", err)
	}
	defer conn.Close()

	prePivotMsg, err := ipc.ReceiveMessage(conn)
	if err != nil {
		return fmt.Errorf("read prepivot message: %w", err)
	}

	slog.Debug(
		"received prepivot message",
		"container_id", c.State.ID,
		"message", prePivotMsg,
	)

	if prePivotMsg != ipc.MsgPrePivot {
		return fmt.Errorf(
			"expected MsgPrePivot ('%b') but got '%b'",
			ipc.MsgPrePivot,
			prePivotMsg,
		)
	}

	slog.Debug("execute createruntime hooks", "container_id", c.State.ID)
	if err := c.execHooks(LifecycleCreateRuntime); err != nil {
		return fmt.Errorf("exec createruntime hooks: %w", err)
	}

	readyMsg, err := ipc.ReceiveMessage(conn)
	if err != nil {
		return fmt.Errorf("read ready message: %w", err)
	}

	slog.Debug(
		"received ready message",
		"container_id", c.State.ID,
		"message", readyMsg,
	)

	switch readyMsg {
	case ipc.MsgInvalidBinary:
		return errors.New("invalid binary")
	case ipc.MsgReady:
		// Let's go!
	default:
		return fmt.Errorf(
			"expected MsgReady ('%b') or MsgInvalidBinary ('%b') but got '%b'",
			ipc.MsgReady,
			ipc.MsgInvalidBinary,
			readyMsg,
		)
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release container process: %w", err)
	}

	c.State.Status = specs.StateCreated
	if err := c.save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}

// Reexec is the entry point for the containerised process. It is responsible
// for setting up the container environment, including namespaces, mounts,
// and security settings, before executing the user-specified process.
func (c *Container) Reexec() error {
	slog.Debug("reexec container", "container_id", c.State.ID)

	// Subsequent syscalls need to happen in a single-threaded context.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	initSockFD := os.Getenv(envInitSockFD)
	if initSockFD == "" {
		return errors.New("missing init sock fd")
	}

	initSockFDVal, err := strconv.Atoi(initSockFD)
	if err != nil {
		return errors.New("invalid init sock fd")
	}

	initConn, err := net.FileConn(
		os.NewFile(uintptr(initSockFDVal), "init_sock_child"),
	)
	if err != nil {
		return err
	}

	containerSockFD := os.Getenv(envContainerSockFD)
	if containerSockFD == "" {
		return errors.New("missing container sock fd")
	}

	containerSockFDVal, err := strconv.Atoi(containerSockFD)
	if err != nil {
		return errors.New("invalid container sock fd")
	}

	listenerFile := os.NewFile(uintptr(containerSockFDVal), "container_sock")
	listener, err := net.FileListener(listenerFile)
	if err != nil {
		return fmt.Errorf("create listener from fd: %w", err)
	}
	listenerFile.Close()
	defer listener.Close()

	slog.Debug("send prepivot message", "container_id", c.State.ID)
	if err := ipc.SendMessage(initConn, ipc.MsgPrePivot); err != nil {
		return fmt.Errorf("failed to send prepivot message: %w", err)
	}

	if err := c.setupPrePivot(); err != nil {
		return err
	}

	if err := c.connectConsole(); err != nil {
		return err
	}

	if err := c.mountConsole(); err != nil {
		return err
	}

	hasMountNamespace := platform.ContainsNSType(
		c.spec.Linux.Namespaces,
		specs.MountNamespace,
	)

	if hasMountNamespace {
		if err := c.pivotRoot(); err != nil {
			return err
		}

		if c.spec.Process != nil {
			if _, err := exec.LookPath(c.spec.Process.Args[0]); err != nil {
				slog.Debug(
					"send invalid binary message",
					"container_id",
					c.State.ID,
				)

				ipc.SendMessage(initConn, ipc.MsgInvalidBinary)
				return fmt.Errorf("find path of user process binary: %w", err)
			}
		}
	}

	slog.Debug("execute createcontainer hooks", "container_id", c.State.ID)
	if err := c.execHooks(LifecycleCreateContainer); err != nil {
		return fmt.Errorf("exec createcontainer hooks: %w", err)
	}

	slog.Debug("send ready message", "container_id", c.State.ID)

	if err := ipc.SendMessage(initConn, ipc.MsgReady); err != nil {
		return fmt.Errorf("failed to send ready message: %w", err)
	}

	initConn.Close()

	containerConn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept on container sock: %w", err)
	}
	defer containerConn.Close()

	startMsg, err := ipc.ReceiveMessage(containerConn)
	if err != nil {
		return fmt.Errorf("read from container sock: %w", err)
	}

	slog.Debug(
		"received start message",
		"container_id", c.State.ID,
		"message", startMsg,
	)

	if startMsg != ipc.MsgStart {
		return fmt.Errorf(
			"expecting MsgStart ('%b') but received '%b'",
			ipc.MsgStart,
			startMsg,
		)
	}

	if c.spec.Process != nil {
		if err := c.setupPostPivot(); err != nil {
			return err
		}
	}

	slog.Debug("execute startcontainer hooks", "container_id", c.State.ID)
	if err := c.execHooks(LifecycleStartContainer); err != nil {
		return fmt.Errorf("exec startcontainer hooks: %w", err)
	}

	if c.spec.Process == nil {
		return nil
	}

	if err := c.execUserProcess(); err != nil {
		return err
	}

	panic("if you got here then something wrong that is not recoverable")
}

// Pause pauses a running container by freezing the cgroup.
func (c *Container) Pause() error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	if !c.canBePaused() {
		return fmt.Errorf(
			"container cannot be paused in current state (%s)",
			c.State.Status,
		)
	}

	if err := platform.FreezeCgroup(c.spec.Linux.CgroupsPath, c.State.ID); err != nil {
		return fmt.Errorf("freeze cgroup: %w", err)
	}

	c.State.Status = specs.ContainerState("paused")

	if err := c.save(); err != nil {
		return fmt.Errorf("save paused state: %w", err)
	}

	return nil
}

// Resume resumes a paused container by thawing the cgroup.
func (c *Container) Resume() error {
	if err := c.Lock(); err != nil {
		return fmt.Errorf("acquire container lock: %w", err)
	}
	defer c.Unlock()

	if err := c.reloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	if !c.canBeResumed() {
		return fmt.Errorf(
			"container cannot be resumed in current state (%s)",
			c.State.Status,
		)
	}

	if err := platform.ThawCgroup(c.spec.Linux.CgroupsPath, c.State.ID); err != nil {
		return fmt.Errorf("thaw cgroup: %w", err)
	}

	c.State.Status = specs.StateRunning

	if err := c.save(); err != nil {
		return fmt.Errorf("save running state: %w", err)
	}

	return nil
}

func (c *Container) configureNamespaces(cmd *exec.Cmd) error {
	cmd.SysProcAttr.Cloneflags = uintptr(0)

	var joinNSParts []string

	for _, ns := range c.spec.Linux.Namespaces {
		if ns.Type == specs.UserNamespace {
			uidMappings, gidMappings := platform.BuildUserNSMappings(
				c.spec.Linux.UIDMappings,
				c.spec.Linux.GIDMappings,
			)

			cmd.SysProcAttr.UidMappings = uidMappings
			cmd.SysProcAttr.GidMappings = gidMappings
			cmd.SysProcAttr.GidMappingsEnableSetgroups = false

			// Explicitly set child to UID/GID 0 so it has necessary permissions to
			// pick up the mapped credentials and capabilities from /proc/<pid>/uid_map
			// and /proc/<pid>/gid_map.
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 0, Gid: 0}
		}

		if ns.Type == specs.TimeNamespace && c.spec.Linux.TimeOffsets != nil {
			if err := platform.SetTimeOffsets(c.spec.Linux.TimeOffsets); err != nil {
				return fmt.Errorf("set timens offsets: %w", err)
			}
		}

		if ns.Path == "" {
			cmd.SysProcAttr.Cloneflags |= platform.NamespaceFlags[ns.Type]
			continue
		}

		if ns.Type == specs.PIDNamespace {
			if err := platform.JoinNS(&ns); err != nil {
				return fmt.Errorf("join pid namespace: %w", err)
			}
			continue
		}

		f, err := platform.OpenNSPath(&ns)
		if err != nil {
			return fmt.Errorf("validate mount ns path: %w", err)
		}

		joinNSParts = append(
			joinNSParts,
			fmt.Sprintf("%s:%s", platform.NamespaceEnvs[ns.Type], f.Name()),
		)
		f.Close()
	}

	if len(joinNSParts) > 0 {
		cmd.Env = append(
			cmd.Env,
			fmt.Sprintf("%s=%s", envJoinNS, strings.Join(joinNSParts, ",")),
		)
	}

	return nil
}

func (c *Container) useTerminal() bool {
	return c.spec.Process != nil &&
		c.spec.Process.Terminal &&
		c.ConsoleSocket != ""
}

func (c *Container) execUserProcess() error {
	if err := os.Chdir(c.spec.Process.Cwd); err != nil {
		return fmt.Errorf("set working directory: %w", err)
	}

	bin, err := exec.LookPath(c.spec.Process.Args[0])
	if err != nil {
		return fmt.Errorf("find path of user process binary: %w", err)
	}

	slog.Debug(
		"execute user process",
		"container_id", c.State.ID,
		"bin", bin,
		"args", c.spec.Process.Args,
	)

	if err := unix.Exec(bin, c.spec.Process.Args, os.Environ()); err != nil {
		return fmt.Errorf(
			"execve (argv0=%s, argv=%s, envv=%v): %w",
			bin, c.spec.Process.Args, os.Environ(), err,
		)
	}

	return nil
}

func (c *Container) mountConsole() error {
	if c.pty == nil || c.spec.Process == nil || !c.spec.Process.Terminal {
		return nil
	}

	target := filepath.Join(c.rootFS(), "dev/console")

	if err := c.pty.MountSlave(target); err != nil {
		return fmt.Errorf("mount slave: %w", err)
	}

	return nil
}

func (c *Container) connectConsole() error {
	if c.ConsoleSocketFD == 0 {
		return nil
	}

	ptmxPath := filepath.Join(c.rootFS(), "dev/pts/ptmx")
	ptsDir := filepath.Join(c.rootFS(), "dev/pts")

	pty, err := terminal.NewPtyAt(ptmxPath, ptsDir)
	if err != nil {
		return fmt.Errorf("new pty: %w", err)
	}

	if c.spec.Process != nil && c.spec.Process.ConsoleSize != nil {
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
	if err := platform.PivotRoot(c.rootFS()); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("change to root directory: %w", err)
	}
	return nil
}

func (c *Container) canBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func (c *Container) canBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}

func (c *Container) canBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateCreated
}

func (c *Container) canBePaused() bool {
	return c.State.Status == specs.StateRunning
}

func (c *Container) canBeResumed() bool {
	return c.State.Status == specs.ContainerState("paused")
}

// rootFS returns the path to the Container root filesystem.
func (c *Container) rootFS() string {
	if strings.HasPrefix(c.spec.Root.Path, "/") {
		return c.spec.Root.Path
	}

	return filepath.Join(c.State.Bundle, c.spec.Root.Path)
}

func (c *Container) stateFilepath() string {
	return filepath.Join(c.RootDir, c.State.ID, "state.json")
}

func (c *Container) reloadState() error {
	slog.Debug("reload container state", "container_id", c.State.ID)

	s, err := os.ReadFile(c.stateFilepath())
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}

	if err := json.Unmarshal(s, c.State); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	return nil
}
