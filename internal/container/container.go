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

// Container represents an OCI container, including its state, specification,
// and other runtime details.
type Container struct {
	State           *specs.State
	ConsoleSocket   string
	ConsoleSocketFD *int

	spec          *specs.Spec
	pty           *terminal.Pty
	pidFile       string
	rootDir       string
	containerSock string
}

// ContainerOpts holds the options for creating a new container.
type ContainerOpts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
	RootDir       string
}

// New creates a container based on the provided opts and saves its state.
// The container will be in the 'creating' state.
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

		containerSock: filepath.Join(
			opts.RootDir,
			opts.ID,
			containerSockFilename,
		),
	}, nil
}

// Save persists the container's state to disk. It creates the required
// directory hierarchy and sets permissions, if needed.
func (c *Container) Save() error {
	containerDir := filepath.Join(c.rootDir, c.State.ID)

	if err := os.MkdirAll(containerDir, 0o755); err != nil {
		return fmt.Errorf("create container directory: %w", err)
	}

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

	if err := os.WriteFile(stateFile, state, 0o644); err != nil {
		return fmt.Errorf("write container state: %w", err)
	}

	if c.pidFile != "" && c.State.Pid > 0 {
		if err := os.WriteFile(
			c.pidFile,
			[]byte(strconv.Itoa(c.State.Pid)),
			0o644,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", c.pidFile, err)
		}
	}

	return nil
}

// Init prepares the container for execution. It executes hooks, sets up the
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
			strconv.Itoa(*c.ConsoleSocketFD),
		)
	}

	if logrus.GetLevel() == logrus.DebugLevel {
		args = append(args, "--debug")
	}

	args = append(args, c.State.ID)

	cmd := exec.Command("/proc/self/exe", args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	initSockParentFD, initSockChildFD, err := ipc.NewSocketPair()
	if err != nil {
		return err
	}

	cmd.ExtraFiles = []*os.File{
		os.NewFile(uintptr(initSockChildFD), "init_sock"),
	}

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
			uidMappings, gidMappings := buildMappings(c.spec)

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
		if platform.IsUnifiedCGroupsMode() {
			if err := platform.AddV2CGroups(
				c.State.ID,
				c.spec.Linux.Resources,
				c.State.Pid,
			); err != nil {
				return fmt.Errorf("add to v2 cgroup: %w", err)
			}
		} else if c.spec.Linux.CgroupsPath != "" {
			if err := platform.AddV1CGroups(
				c.spec.Linux.CgroupsPath,
				c.spec.Linux.Resources,
				c.State.Pid,
			); err != nil {
				return fmt.Errorf("add to v1 cgroup: %w", err)
			}
		}
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release container process: %w", err)
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
		return fmt.Errorf("expecting 'ready' but received '%s'", msg)
	}

	c.State.Status = specs.StateCreated
	if err := c.Save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}

// Reexec is the entry point for the containerised process. It is responsible
// for setting up the container's environment, including namespaces, mounts,
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

	// 3 is first extra file, after 0=stdin 1=stdout 2=stderr.
	conn, err := ipc.FDToConn(3)
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

	panic("if you got here then something went horribly wrong")
}

// Start begins the execution of the container. It executes pre-start and
// post-start hooks and sends the "start" message to the runtime process.
func (c *Container) Start() error {
	if c.spec.Process == nil {
		c.State.Status = specs.StateStopped
		if err := c.Save(); err != nil {
			return err
		}
		// nothing to do; silent return
		return nil
	}

	if !c.canBeStarted() {
		return fmt.Errorf(
			"container cannot be started in current state (%s)",
			c.State.Status,
		)
	}

	if err := c.execHooks(LifecyclePrestart); err != nil {
		logrus.Errorf("failed to exec prestart hooks: %s", err)
		return fmt.Errorf("execute prestart hooks: %w", err)
	}

	containerSock := ipc.NewSocket(c.containerSock)

	conn, err := containerSock.Dial()
	if err != nil {
		logrus.Errorf("failed to dial container sock: %s", err)
		return fmt.Errorf("dial container sock: %w", err)
	}

	if err := ipc.SendMessage(conn, startMsg); err != nil {
		logrus.Errorf("failed to write start to sock: %s", err)
		return fmt.Errorf("write 'start' msg to container sock: %w", err)
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

// Delete removes the container from the system. If force is true then it will
// delete the container, regardless of the container's state.
func (c *Container) Delete(force bool) error {
	if !force && !c.canBeDeleted() {
		return fmt.Errorf(
			"container cannot be deleted in current state (%s) try using '--force'",
			c.State.Status,
		)
	}

	if c.spec.Linux.Resources != nil {
		if platform.IsUnifiedCGroupsMode() {
			if err := platform.DeleteV2CGroups(c.State.ID); err != nil {
				return err
			}
		} else if c.spec.Linux.CgroupsPath != "" {
			if err := platform.DeleteV1CGroups(c.spec.Linux.CgroupsPath); err != nil {
				return err
			}
		}
	} else if c.State.Pid != 0 {
		if err := syscall.Kill(
			c.State.Pid,
			syscall.SIGKILL,
		); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("send kill signal to process: %w", err)
		}
	}

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

// Kill sends a signal to the container's process and executes post-stop hooks.
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

	c.State.Status = specs.StateStopped
	if err := c.Save(); err != nil {
		return fmt.Errorf("save stopped state: %w", err)
	}

	if err := c.execHooks(LifecyclePoststop); err != nil {
		fmt.Printf("Warning: failed to exec poststop hooks: %s\n", err)
	}

	return nil
}

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
		return errors.New("process is required")
	}

	if err := platform.PivotRoot(c.rootFS()); err != nil {
		return err
	}

	return nil
}

func (c *Container) connectConsole() error {
	if c.ConsoleSocketFD == nil {
		return nil
	}

	pty, err := terminal.NewPty()
	if err != nil {
		return fmt.Errorf("new pty: %w", err)
	}

	if c.spec.Process.ConsoleSize != nil {
		unix.IoctlSetWinsize(
			int(pty.Slave.Fd()),
			unix.TIOCSWINSZ,
			&unix.Winsize{
				Row: uint16(c.spec.Process.ConsoleSize.Height),
				Col: uint16(c.spec.Process.ConsoleSize.Width),
			},
		)
	}

	if err := terminal.SendPty(*c.ConsoleSocketFD, pty); err != nil {
		return fmt.Errorf("connect pty and socket: %w", err)
	}

	if err := pty.Connect(); err != nil {
		return fmt.Errorf("connect pty: %w", err)
	}

	c.pty = pty

	return nil
}

func (c *Container) mountConsole() error {
	if c.ConsoleSocketFD == nil || !c.spec.Process.Terminal {
		return nil
	}

	target := filepath.Join(c.rootFS(), "dev/console")

	if err := c.pty.MountSlave(target); err != nil {
		return err
	}

	return nil
}

func (c *Container) waitStart() error {
	containerSock := ipc.NewSocket(c.containerSock)
	listener, err := containerSock.Listen()
	if err != nil {
		return fmt.Errorf("listen on container sock: %w", err)
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		logrus.Errorf("failed to accept on container socket: %s", err)
		return fmt.Errorf("accept on container sock: %w", err)
	}
	defer conn.Close()

	msg, err := ipc.ReceiveMessage(conn)
	if err != nil {
		return fmt.Errorf("read from container sock: %w", err)
	}
	if msg != startMsg {
		return fmt.Errorf("expecting 'start' but received '%s'", msg)
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

	if err := syscall.Exec(bin, c.spec.Process.Args, os.Environ()); err != nil {
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
		if err := syscall.Sethostname([]byte(c.spec.Hostname)); err != nil {
			return fmt.Errorf("set hostname: %w", err)
		}

		if err := syscall.Setdomainname([]byte(c.spec.Domainname)); err != nil {
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

func (c *Container) useTerminal() bool {
	return c.spec.Process != nil &&
		c.spec.Process.Terminal &&
		c.ConsoleSocket != ""
}

// Load retrieves an existing container with the given id at the given rootDir.
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

// Exists checks if a container with the given id at the given rootDir exists.
func Exists(id, rootDir string) bool {
	_, err := os.Stat(filepath.Join(rootDir, id))

	return err == nil
}

func buildMappings(
	spec *specs.Spec,
) ([]syscall.SysProcIDMap, []syscall.SysProcIDMap) {
	uidMappings := make([]syscall.SysProcIDMap, 0, 1)
	gidMappings := make([]syscall.SysProcIDMap, 0, 1)

	if uidCount := len(spec.Linux.UIDMappings); uidCount > 0 {
		uidMappings = slices.Grow(uidMappings, uidCount-1)

		for _, m := range spec.Linux.UIDMappings {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: int(m.ContainerID),
				HostID:      int(m.HostID),
				Size:        int(m.Size),
			})
		}
	} else {
		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: 0,
			HostID:      os.Getuid(),
			Size:        1,
		})
	}

	if gidCount := len(spec.Linux.GIDMappings); gidCount > 0 {
		for _, m := range spec.Linux.GIDMappings {
			gidMappings = slices.Grow(gidMappings, gidCount-1)

			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: int(m.ContainerID),
				HostID:      int(m.HostID),
				Size:        int(m.Size),
			})
		}
	} else {
		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: 0,
			HostID:      os.Getgid(),
			Size:        1,
		})
	}

	return uidMappings, gidMappings
}
