// Package container provides functionality for creating, running, and managing
// OCI-compliant containers.
package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/nixpig/anocir/internal/container/hooks"
	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// lockFilename is the filename of the lockfile used to synchronise access
	// to container operations.
	lockFilename = "c.lock"

	// envInitSockFD is the name of the environment variable used to pass the
	// init sock file descriptor.
	envInitSockFD = "_ANOCIR_INIT_SOCK_FD"

	// containerSockFilename is the filename of the socket used by the runtime to
	// send messages to the container.
	containerSockFilename = "c.sock"

	// readyMsg is the message sent over the init socketpair when the container
	// is created and ready to receive commands.
	readyMsg = "ready"
	// startMsg is the message sent on the container socket to start the created
	// container.
	startMsg = "start"
)

var (
	// ErrOperationInProgress is returned when the container is locked by another
	// operation.
	ErrOperationInProgress = errors.New("operation already in progress")

	// ErrMissingProcess is returned when the provided spec is missing a process.
	ErrMissingProcess = errors.New("process is required")
)

// Container represents an OCI container, including its state, specification,
// and other runtime details.
type Container struct {
	State           *specs.State
	ConsoleSocket   string
	ConsoleSocketFD int

	spec          *specs.Spec
	pty           *terminal.Pty
	pidFile       string
	rootDir       string
	containerSock string
	logFile       string
	lockFile      *os.File
}

// ContainerOpts holds the options for creating a new Container.
type ContainerOpts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
	RootDir       string
	LogFile       string
}

// New creates a Container based on the provided opts and saves its state.
// The Container will be in the 'creating' state.
func New(opts *ContainerOpts) (*Container, error) {
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

		rootDir: opts.RootDir,
		logFile: opts.LogFile,
		containerSock: filepath.Join(
			opts.RootDir,
			opts.ID,
			containerSockFilename,
		),
	}, nil
}

// Save persists the Container state to disk. It creates the required directory
// hierarchy and sets the needed permissions.
func (c *Container) Save() error {
	containerDir := filepath.Join(c.rootDir, c.State.ID)

	if c.spec.Linux != nil &&
		len(c.spec.Linux.UIDMappings) > 0 &&
		len(c.spec.Linux.GIDMappings) > 0 {
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

	stateFile := filepath.Join(containerDir, "state.json")

	if err := platform.AtomicWriteFile(stateFile, state, 0o644); err != nil {
		return fmt.Errorf("write container state: %w", err)
	}

	if c.pidFile != "" && c.State.Pid > 0 {
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

// Init prepares the Container for execution. It executes hooks, sets up the
// terminal if necessary, and re-execs the runtime binary to containerise the
// process.
func (c *Container) Init() error {
	if err := c.execHooks(LifecycleCreateRuntime); err != nil {
		return fmt.Errorf("exec createruntime hooks: %w", err)
	}

	if err := c.execHooks(LifecycleCreateContainer); err != nil {
		return fmt.Errorf("exec createcontainer hooks: %w", err)
	}

	args := []string{"reexec", "--root", c.rootDir}

	if c.useTerminal() {
		consoleSocketFD, err := terminal.Setup(c.rootFS(), c.ConsoleSocket)
		if err != nil {
			return err
		}

		c.ConsoleSocketFD = consoleSocketFD
		args = append(
			args,
			"--console-socket-fd",
			strconv.Itoa(c.ConsoleSocketFD),
		)
	}

	if logrus.GetLevel() == logrus.DebugLevel {
		args = append(args, "--debug")
	}

	args = append(args, "--log", c.logFile)
	args = append(args, c.State.ID)

	cmd := exec.Command("/proc/self/exe", args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	initSockParentFD, initSockChildFD, err := ipc.NewSocketPair()
	if err != nil {
		return err
	}

	initSockFile := os.NewFile(uintptr(initSockChildFD), "init_sock")
	cmd.ExtraFiles = []*os.File{initSockFile}

	cmd.Env = append(
		cmd.Env,
		fmt.Sprintf(
			"%s=%d",
			envInitSockFD,
			slices.Index(cmd.ExtraFiles, initSockFile)+3,
		),
	)

	if c.spec.Process != nil && c.spec.Process.OOMScoreAdj != nil {
		if err := platform.AdjustOOMScore(
			*c.spec.Process.OOMScoreAdj,
		); err != nil {
			return fmt.Errorf("adjust oom score: %w", err)
		}
	}

	cloneFlags := uintptr(0)

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
			cloneFlags |= platform.NamespaceFlags[ns.Type]
		} else {
			if err := platform.ValidateNSPath(&ns); err != nil {
				return fmt.Errorf("validate ns path: %w", err)
			}

			if ns.Type == specs.MountNamespace {
				// Mount namespaces do not work across OS threads and Go cannot
				// guarantee what thread any newly spawned goroutines will land on,
				// so this needs to be done in single-threaded context in C before the
				// reexec.
				gonsEnv := fmt.Sprintf(
					"gons_%s=%s",
					platform.NamespaceEnvs[ns.Type],
					ns.Path,
				)
				cmd.Env = append(cmd.Env, gonsEnv)
			} else {
				if err := platform.SetNS(ns.Path); err != nil {
					return fmt.Errorf("set namespace: %w", err)
				}
			}
		}
	}

	cmd.SysProcAttr.Cloneflags = cloneFlags

	if c.spec.Process != nil && c.spec.Process.Env != nil {
		cmd.Env = append(cmd.Env, c.spec.Process.Env...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reexec container process: %w", err)
	}

	unix.Close(initSockChildFD)

	c.State.Pid = cmd.Process.Pid

	if err := c.Save(); err != nil {
		return fmt.Errorf("save container state: %w", err)
	}

	if c.spec.Linux.Resources != nil {
		if err := platform.AddCGroups(c.State, c.spec); err != nil {
			return fmt.Errorf("create cgroups: %w", err)
		}
	}

	conn, err := ipc.FDToConn(initSockParentFD)
	if err != nil {
		return fmt.Errorf("accept on init sock parent: %w", err)
	}
	defer conn.Close()

	msg, err := ipc.ReceiveMessage(conn)
	if err != nil {
		return err
	}
	if msg != readyMsg {
		return fmt.Errorf("expecting '%s' but received '%s'", readyMsg, msg)
	}

	if err := cmd.Process.Release(); err != nil {
		// TODO: Cleanup cgroups and the like.
		return fmt.Errorf("release container process: %w", err)
	}

	c.State.Status = specs.StateCreated
	if err := c.Save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}

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

// Start begins the execution of the Container. It executes pre-start and
// post-start hooks and sends the "start" message to the runtime process.
func (c *Container) Start() error {
	if c.spec.Process == nil {
		c.State.Status = specs.StateStopped
		if err := c.Save(); err != nil {
			return fmt.Errorf("save state stopped: %w", err)
		}
		// Nothing to do; silent return.
		return nil
	}

	if !c.canBeStarted() {
		return fmt.Errorf(
			"container cannot be started in current state (%s)",
			c.State.Status,
		)
	}

	if err := c.execHooks(LifecyclePrestart); err != nil {
		return fmt.Errorf("execute prestart hooks: %w", err)
	}

	containerSock := ipc.NewSocket(c.containerSock)

	conn, err := containerSock.Dial()
	if err != nil {
		return fmt.Errorf("dial container sock: %w", err)
	}

	if err := ipc.SendMessage(conn, startMsg); err != nil {
		return fmt.Errorf("write '%s' msg to container sock: %w", startMsg, err)
	}
	defer conn.Close()

	c.State.Status = specs.StateRunning
	if err := c.Save(); err != nil {
		return fmt.Errorf("save state running: %w", err)
	}

	if err := c.execHooks(LifecyclePoststart); err != nil {
		return fmt.Errorf("exec poststart hooks: %w", err)
	}

	return nil
}

// Delete removes the Container from the system. If force is true then it will
// delete the Container, regardless of the its state.
func (c *Container) Delete(force bool) error {
	if !force && !c.canBeDeleted() {
		return fmt.Errorf(
			"container cannot be deleted in current state (%s) try using '--force'",
			c.State.Status,
		)
	}

	if c.spec.Linux.Resources != nil {
		if err := platform.DeleteCGroups(c.State, c.spec); err != nil {
			return fmt.Errorf("delete cgroups: %w", err)
		}
	} else if c.State.Pid != 0 {
		if err := unix.Kill(
			c.State.Pid,
			unix.SIGKILL,
		); err != nil && !errors.Is(err, unix.ESRCH) {
			return fmt.Errorf("send kill signal to process: %w", err)
		}

		unix.Wait4(c.State.Pid, nil, 0, nil)
	}

	// TODO: Review whether need to remove pidfile.

	if err := os.RemoveAll(
		filepath.Join(c.rootDir, c.State.ID),
	); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	if err := c.execHooks(LifecyclePoststop); err != nil {
		fmt.Printf("Warning: failed to exec poststop hooks: %s\n", err)
	}

	return nil
}

// Kill sends the given sig to the Container process and executes post-stop
// hooks.
func (c *Container) Kill(sig string) error {
	if !c.canBeKilled() {
		return fmt.Errorf(
			"container cannot be killed in current state (%s)",
			c.State.Status,
		)
	}

	if err := platform.SendSignal(
		c.State.Pid,
		platform.ParseSignal(sig),
	); err != nil {
		return fmt.Errorf(
			"send signal '%s' to process '%d': %w",
			sig,
			c.State.Pid,
			err,
		)
	}

	// TODO: Wait for signal to be handled. Signal isn't necessarily going to
	// stop the process, so only update state and exec hooks if needed.

	c.State.Status = specs.StateStopped
	if err := c.Save(); err != nil {
		return fmt.Errorf("save stopped state: %w", err)
	}

	if err := c.execHooks(LifecyclePoststop); err != nil {
		fmt.Printf("Warning: failed to exec poststop hooks: %s\n", err)
	}

	return nil
}

// GetState returns the state of the container. In the case the container
// process no longer exists, it has the side effect of internally modifying
// the state to be 'stopped' before returning.
func (c *Container) GetState() (string, error) {
	if c.State.Pid != 0 {
		process, err := os.FindProcess(c.State.Pid)
		if err != nil {
			return "", fmt.Errorf("find container process: %w", err)
		}

		if err := process.Signal(unix.Signal(0)); err != nil {
			c.State.Status = specs.StateStopped
			if err := c.Save(); err != nil {
				return "", fmt.Errorf("save stopped state: %w", err)
			}
		}
	}

	state, err := json.Marshal(c.State)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	return string(state), nil
}

// execHooks executes the hooks for the given phase of the Container execution.
func (c *Container) execHooks(phase Lifecycle) error {
	if c.spec.Hooks == nil {
		return nil
	}

	var h []specs.Hook

	switch phase {
	case LifecycleCreateRuntime:
		h = append(h, c.spec.Hooks.CreateRuntime...)
	case LifecycleCreateContainer:
		h = append(h, c.spec.Hooks.CreateContainer...)
	case LifecycleStartContainer:
		h = append(h, c.spec.Hooks.StartContainer...)
	case LifecyclePrestart:
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker.
		h = append(h, c.spec.Hooks.Prestart...)
	case LifecyclePoststart:
		h = append(h, c.spec.Hooks.Poststart...)
	case LifecyclePoststop:
		h = append(h, c.spec.Hooks.Poststop...)
	}

	if len(h) > 0 {
		if err := hooks.ExecHooks(h, c.State); err != nil {
			return err
		}
	}

	return nil
}

// rootFS determines and returns the root filesystem.
func (c *Container) rootFS() string {
	if strings.HasPrefix(c.spec.Root.Path, "/") {
		return c.spec.Root.Path
	}

	return filepath.Join(c.State.Bundle, c.spec.Root.Path)
}

func (c *Container) canBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}

func (c *Container) canBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func (c *Container) canBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateCreated
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

func (c *Container) Lock() error {
	lockPath := filepath.Join(c.rootDir, c.State.ID, lockFilename)
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

func (c *Container) Unlock() error {
	if c.lockFile == nil {
		return nil
	}

	defer c.lockFile.Close()
	return unix.Flock(int(c.lockFile.Fd()), unix.LOCK_UN)
}

func (c *Container) ReloadState() error {
	stateFile := filepath.Join(c.rootDir, c.State.ID, "state.json")
	s, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}

	if err := json.Unmarshal(s, c.State); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	return nil
}

func (c *Container) useTerminal() bool {
	return c.spec.Process != nil &&
		c.spec.Process.Terminal &&
		c.ConsoleSocket != ""
}

// Load retrieves an existing Container with the given id at the given rootDir.
func Load(id, rootDir string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(rootDir, id, "state.json"))
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state *specs.State
	if err := json.Unmarshal(s, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	config, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(config, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	c := &Container{
		State:         state,
		spec:          spec,
		rootDir:       rootDir,
		containerSock: filepath.Join(rootDir, id, containerSockFilename),
	}

	return c, nil
}

// WithLock loads a Container with the given id at the given
// rootDir, acquires an exclusive lock, refreshes the state, and executes the
// given fn, finally releasing the lock.
func WithLock(id, rootDir string, fn func(*Container) error) error {
	c, err := Load(id, rootDir)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := c.Lock(); err != nil {
		return fmt.Errorf("lock access to container: %w", err)
	}
	defer c.Unlock()

	if err := c.ReloadState(); err != nil {
		return fmt.Errorf("reload container state: %w", err)
	}

	return fn(c)
}

// Exists checks if a container exists with the given id at the given rootDir.
func Exists(id, rootDir string) bool {
	_, err := os.Stat(filepath.Join(rootDir, id))

	return err == nil
}
