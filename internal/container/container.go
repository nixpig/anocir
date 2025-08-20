// Package container provides functionality for creating, running, and managing
// OCI-compliant containers.
package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nixpig/anocir/internal/anosys"
	"github.com/nixpig/anocir/internal/hooks"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// TODO: make this configuration-driven
var containerRootDir = "/var/lib/anocir/containers"

const (
	initSockFilename      = "init.sock"
	containerSockFilename = "container.sock"
)

type Lifecycle string

const (
	LifecycleCreateRuntime   Lifecycle = "createRuntime"
	LifecycleCreateContainer Lifecycle = "createContainer"
	LifecycleStartContainer  Lifecycle = "startContainer"
	LifecyclePrestart        Lifecycle = "prestart"
	LifecyclePoststart       Lifecycle = "poststart"
	LifecyclePoststop        Lifecycle = "poststop"
)

// Container represents an OCI container, including its state, specification,
// and other runtime details.
type Container struct {
	State           *specs.State
	Spec            *specs.Spec
	ConsoleSocket   string
	ConsoleSocketFD *int
	PIDFile         string
	Opts            *NewContainerOpts
}

// NewContainerOpts holds the options for creating a new container.
type NewContainerOpts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
}

// Initialiser functions are passed to init and reexec functions of a
// Container, typically to perform syscalls and other initialisation tasks.
type Initialiser func(spec *specs.Spec, rootfs string) error

// New creates a new container instance based on the provided options.
func New(opts *NewContainerOpts) (*Container, error) {
	state := &specs.State{
		Version:     specs.Version,
		ID:          opts.ID,
		Bundle:      opts.Bundle,
		Annotations: opts.Spec.Annotations,
		Status:      specs.StateCreating,
	}

	c := &Container{
		State:         state,
		Spec:          opts.Spec,
		ConsoleSocket: opts.ConsoleSocket,
		PIDFile:       opts.PIDFile,
		Opts:          opts,
	}

	if err := c.Save(); err != nil {
		return nil, fmt.Errorf("save created state: %w", err)
	}

	return c, nil
}

// Save persists the container's state.
func (c *Container) Save() error {
	if err := os.MkdirAll(
		filepath.Join(containerRootDir, c.State.ID),
		0666,
	); err != nil {
		return fmt.Errorf("create container directory: %w", err)
	}

	state, err := json.Marshal(c.State)
	if err != nil {
		return fmt.Errorf("serialise container state: %w", err)
	}

	if err := os.WriteFile(
		filepath.Join(containerRootDir, c.State.ID, "state.json"),
		state,
		0666,
	); err != nil {
		return fmt.Errorf("write container state: %w", err)
	}

	if c.PIDFile != "" {
		if err := os.WriteFile(
			c.PIDFile,
			[]byte(strconv.Itoa(c.State.Pid)),
			0666,
		); err != nil {
			return fmt.Errorf(
				"write container PID to file (%s): %w",
				c.PIDFile,
				err,
			)
		}
	}

	return nil
}

// Init prepares the container for execution. It executes hooks,
// sets up the terminal if necessary, and re-execs the runtime binary to
// containerise the process.
func (c *Container) Init() error {
	return c.init()
}

func (c *Container) init() error {
	if err := c.execHook(LifecycleCreateRuntime); err != nil {
		return fmt.Errorf("exec createruntime hooks: %w", err)
	}

	if err := c.execHook(LifecycleCreateContainer); err != nil {
		return fmt.Errorf("exec createcontainer hooks: %w", err)
	}

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		c.ConsoleSocket != ""

	if useTerminal {
		var err error
		if c.ConsoleSocketFD, err = terminal.Setup(
			c.rootFS(),
			c.ConsoleSocket,
		); err != nil {
			return err
		}
	}

	args := []string{"reexec"}

	logLevel := logrus.GetLevel()
	if logLevel == logrus.DebugLevel {
		args = append(args, "--debug")
	}

	if c.ConsoleSocketFD != nil {
		fd := strconv.Itoa(*c.ConsoleSocketFD)
		args = append(args, "--console-socket-fd", fd)
	}

	args = append(args, c.State.ID)

	cmd := exec.Command("/proc/self/exe", args...)

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, initSockFilename),
	)
	if err != nil {
		return fmt.Errorf("listen on init sock: %w", err)
	}
	defer listener.Close()

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := anosys.AdjustOOMScore(
			*c.Spec.Process.OOMScoreAdj,
		); err != nil {
			return fmt.Errorf("adjust oom score: %w", err)
		}
	}

	cloneFlags := uintptr(0)

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap

	for _, ns := range c.Spec.Linux.Namespaces {
		if ns.Type == specs.UserNamespace {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			})

			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			})
		}

		if ns.Type == specs.TimeNamespace {
			if c.Spec.Linux.TimeOffsets != nil {
				if err := anosys.SetTimeOffsets(
					c.Spec.Linux.TimeOffsets,
				); err != nil {
					return fmt.Errorf("set timens offsets: %w", err)
				}
			}
		}

		if ns.Path == "" {
			cloneFlags |= anosys.NamespaceFlags[ns.Type]
		} else {
			suffix := fmt.Sprintf(
				"/%s",
				anosys.NamespaceEnvs[ns.Type],
			)
			if !strings.HasSuffix(ns.Path, suffix) &&
				ns.Type != specs.PIDNamespace {
				return fmt.Errorf(
					"namespace type (%s) and path (%s) do not match",
					ns.Type,
					ns.Path,
				)
			}

			if ns.Type == specs.MountNamespace {
				// mount namespaces do not work across threads, so this needs to be done
				// in single-threaded context in C before the reexec
				gonsEnv := fmt.Sprintf(
					"gons_%s=%s",
					anosys.NamespaceEnvs[ns.Type],
					ns.Path,
				)
				cmd.Env = append(cmd.Env, gonsEnv)
			} else {
				fd, err := syscall.Open(ns.Path, syscall.O_RDONLY, 0666)
				if err != nil {
					return fmt.Errorf("open ns path: %w", err)
				}

				_, _, errno := syscall.Syscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
				if errno != 0 {
					return fmt.Errorf("errno: %w", errno)
				}

				syscall.Close(fd)
			}
		}
	}

	// FIXME: needed to run 'linux_uid_mappings'
	// for _, m := range c.Spec.Linux.UIDMappings {
	// 	uidMappings = append(uidMappings, syscall.SysProcIDMap{
	// 		ContainerID: int(m.ContainerID),
	// 		HostID:      int(m.HostID),
	// 		Size:        int(m.Size),
	// 	})
	// }
	//
	// for _, m := range c.Spec.Linux.GIDMappings {
	// 	gidMappings = append(gidMappings, syscall.SysProcIDMap{
	// 		ContainerID: int(m.ContainerID),
	// 		HostID:      int(m.HostID),
	// 		Size:        int(m.Size),
	// 	})
	// }

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  cloneFlags,
		UidMappings: uidMappings,
		GidMappings: gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		cmd.Env = append(cmd.Env, c.Spec.Process.Env...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reexec container process: %w", err)
	}

	c.State.Pid = cmd.Process.Pid
	if err := c.Save(); err != nil {
		return fmt.Errorf("save container pid state: %w", err)
	}

	if c.Spec.Linux.Resources != nil {
		if anosys.IsUnifiedCGroupsMode() {
			if err := anosys.AddV2CGroups(
				c.State.ID,
				c.Spec.Linux.Resources,
				c.State.Pid,
			); err != nil {
				return err
			}
		} else if c.Spec.Linux.CgroupsPath != "" {
			if err := anosys.AddV1CGroups(
				c.Spec.Linux.CgroupsPath,
				c.Spec.Linux.Resources,
				c.State.Pid,
			); err != nil {
				return err
			}
		}
	}

	if err := cmd.Process.Release(); err != nil {
		logrus.Errorf("failed to release container process: %s", err)
		return fmt.Errorf("release container process: %w", err)
	}

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept on init sock: %w", err)
	}
	defer conn.Close()

	b := make([]byte, 128)
	n, err := conn.Read(b)
	if err != nil {
		return fmt.Errorf("read bytes from init sock connection: %w", err)
	}

	msg := string(b[:n])
	if msg != "ready" {
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
	return c.reexec(
		[]Initialiser{prePivotInitialisers},
		[]Initialiser{postPivotInitialisers},
	)
}

func (c *Container) reexec(
	prePivotIntialisers,
	postPivotInitialisers []Initialiser,
) error {
	// subsequent syscalls need to happen in a single-threaded context
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var err error
	var pty *terminal.Pty
	if c.ConsoleSocketFD != nil {
		pty, err = connectConsole(c.Spec, *c.ConsoleSocketFD)
		if err != nil {
			return err
		}
	}

	for _, initialiser := range prePivotIntialisers {
		if err := initialiser(c.Spec, c.rootFS()); err != nil {
			return err
		}
	}

	if c.ConsoleSocketFD != nil && c.Spec.Process.Terminal {
		if err := mountConsole(c.rootFS(), pty); err != nil {
			return err
		}
	}

	if err := notifyReady(c.State.ID); err != nil {
		return err
	}

	if err := waitStart(c.State.ID); err != nil {
		return err
	}

	if err := c.pivotRoot(); err != nil {
		return err
	}

	for _, initialiser := range postPivotInitialisers {
		if err := initialiser(c.Spec, c.rootFS()); err != nil {
			return err
		}
	}

	if err := c.execHook(LifecycleStartContainer); err != nil {
		return fmt.Errorf("exec startcontainer hooks: %w", err)
	}

	if err := execUserProcess(c.Spec); err != nil {
		return err
	}

	panic("if you got here then something went horribly wrong")
}

// Start begins the execution of the container. It executes pre-start and
// post-start hooks and sends the "start" message to the runtime process.
func (c *Container) Start() error {
	if c.Spec.Process == nil {
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

	if err := c.execHook(LifecyclePrestart); err != nil {
		logrus.Errorf("failed to exec prestart hooks: %s", err)
		return fmt.Errorf("execute prestart hooks: %w", err)
	}

	conn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, containerSockFilename),
	)
	if err != nil {
		logrus.Errorf("failed to dial container sock: %s", err)
		return fmt.Errorf("dial container sock: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		logrus.Errorf("failed to write start to sock: %s", err)
		return fmt.Errorf("write 'start' msg to container sock: %w", err)
	}
	conn.Close()

	c.State.Status = specs.StateRunning
	if err := c.Save(); err != nil {
		return fmt.Errorf("save state running: %w", err)
	}

	if err := c.execHook(LifecyclePoststart); err != nil {
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

	process, err := os.FindProcess(c.State.Pid)
	if err != nil {
		return fmt.Errorf("find container process to delete: %w", err)
	}
	if process != nil {
		process.Signal(unix.SIGKILL)
	}

	if err := os.RemoveAll(
		filepath.Join(containerRootDir, c.State.ID),
	); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	if err := c.execHook(LifecyclePoststop); err != nil {
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

	if err := anosys.SendSignal(c.State.Pid, sig); err != nil {
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

	if err := c.execHook(LifecyclePoststop); err != nil {
		fmt.Println("Warning: failed to execute poststop hooks")
	}

	return nil
}

func (c *Container) execHook(phase Lifecycle) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var h []specs.Hook

	switch phase {
	case LifecycleCreateRuntime:
		h = append(h, c.Spec.Hooks.CreateRuntime...)
	case LifecycleCreateContainer:
		h = append(h, c.Spec.Hooks.CreateContainer...)
	case LifecycleStartContainer:
		h = append(h, c.Spec.Hooks.StartContainer...)
	case LifecyclePrestart:
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker
		h = append(h, c.Spec.Hooks.Prestart...)
	case LifecyclePoststart:
		h = append(h, c.Spec.Hooks.Poststart...)
	case LifecyclePoststop:
		h = append(h, c.Spec.Hooks.Poststop...)
	}

	if len(h) > 0 {
		if err := hooks.ExecHooks(h, c.State); err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) rootFS() string {
	if strings.HasPrefix(c.Spec.Root.Path, "/") {
		return c.Spec.Root.Path
	}

	return filepath.Join(c.State.Bundle, c.Spec.Root.Path)
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
	if c.Spec.Process == nil {
		return errors.New("process is required")
	}

	logrus.Debug("pivot root")
	if err := anosys.PivotRoot(c.rootFS()); err != nil {
		return err
	}
	logrus.Debug("pivoted root!")

	return nil
}

// Load retrieves a container's state and specification, based on its ID.
func Load(id string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(containerRootDir, id, "state.json"))
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
		State: state,
		Spec:  spec,
	}

	if err := c.Save(); err != nil {
		return nil, fmt.Errorf("save state: %w", err)
	}

	return c, nil
}

// Exists checks if a container with the given ID exists.
func Exists(containerID string) bool {
	_, err := os.Stat(filepath.Join(containerRootDir, containerID))

	return err == nil
}

func prePivotInitialisers(spec *specs.Spec, rootfs string) error {
	if err := anosys.MountRootfs(rootfs); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	if err := anosys.MountProc(rootfs); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	spec.Mounts = append(spec.Mounts, specs.Mount{
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

	if err := anosys.MountSpecMounts(spec.Mounts, rootfs); err != nil {
		return fmt.Errorf("mount spec: %w", err)
	}

	if err := anosys.MountDefaultDevices(rootfs); err != nil {
		return fmt.Errorf("mount default devices: %w", err)
	}

	if err := anosys.CreateDeviceNodes(
		spec.Linux.Devices,
		rootfs,
	); err != nil {
		return fmt.Errorf("mount devices from spec: %w", err)
	}

	if err := anosys.CreateDefaultSymlinks(rootfs); err != nil {
		return fmt.Errorf("create default symlinks: %w", err)
	}

	return nil
}

func postPivotInitialisers(spec *specs.Spec, rootfs string) error {
	if spec.Linux.Sysctl != nil {
		if err := anosys.SetSysctl(spec.Linux.Sysctl); err != nil {
			return fmt.Errorf("set sysctl: %w", err)
		}
	}

	if err := anosys.MountMaskedPaths(
		spec.Linux.MaskedPaths,
	); err != nil {
		return err
	}

	if err := anosys.MountReadonlyPaths(
		spec.Linux.ReadonlyPaths,
	); err != nil {
		return err
	}

	if err := anosys.SetRootfsMountPropagation(
		spec.Linux.RootfsPropagation,
	); err != nil {
		return err
	}

	if spec.Root.Readonly {
		if err := anosys.MountRootReadonly(); err != nil {
			return err
		}
	}

	hasUTSNamespace := slices.ContainsFunc(
		spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	)

	if hasUTSNamespace {
		if err := syscall.Sethostname([]byte(spec.Hostname)); err != nil {
			return err
		}

		if err := syscall.Setdomainname([]byte(spec.Domainname)); err != nil {
			return err
		}
	}

	if err := anosys.SetRlimits(spec.Process.Rlimits); err != nil {
		return fmt.Errorf("set rlimits: %w", err)
	}

	if spec.Process.Capabilities != nil {
		if err := anosys.SetCapabilities(spec.Process.Capabilities); err != nil {
			return fmt.Errorf("set capabilities: %w", err)
		}
	}

	if spec.Process.NoNewPrivileges {
		if err := anosys.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	if spec.Process.Scheduler != nil {
		if err := anosys.SetSchedAttrs(spec.Process.Scheduler); err != nil {
			return fmt.Errorf("set sched attrs: %w", err)
		}
	}

	if spec.Process.IOPriority != nil {
		if err := anosys.SetIOPriority(spec.Process.IOPriority); err != nil {
			return fmt.Errorf("set ioprio: %w", err)
		}
	}

	if err := anosys.SetUser(&spec.Process.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	return nil
}

func connectConsole(spec *specs.Spec, fd int) (*terminal.Pty, error) {
	pty, err := terminal.NewPty()
	if err != nil {
		return nil, fmt.Errorf("new pty: %w", err)
	}

	if spec.Process.ConsoleSize != nil {
		unix.IoctlSetWinsize(
			int(pty.Slave.Fd()),
			unix.TIOCSWINSZ,
			&unix.Winsize{
				Row: uint16(spec.Process.ConsoleSize.Height),
				Col: uint16(spec.Process.ConsoleSize.Width),
			},
		)
	}

	if err := terminal.SendPty(
		fd,
		pty,
	); err != nil {
		return nil, fmt.Errorf("connect pty and socket: %w", err)
	}

	if err := pty.Connect(); err != nil {
		return nil, fmt.Errorf("connect pty: %w", err)
	}

	return pty, nil
}

func mountConsole(rootfs string, pty *terminal.Pty) error {
	if err := pty.MountSlave(filepath.Join(rootfs, "dev/pts/0")); err != nil {
		return err
	}
	if err := os.Symlink("/dev/pts/0", filepath.Join(rootfs, "dev/console")); err != nil {
		return fmt.Errorf("create console symlink: %w", err)
	}

	return nil
}

func notifyReady(id string) error {
	// wait a sec for init sock to be ready before dialing - this is nasty
	// TODO: use file lock to synchronise
	for range 10 {
		if _, err := os.Stat(filepath.Join(
			containerRootDir,
			id,
			initSockFilename,
		)); errors.Is(err, os.ErrNotExist) {
			time.Sleep(time.Millisecond * 100)
			continue
		}
	}

	initConn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, id, initSockFilename),
	)
	if err != nil {
		return fmt.Errorf("dial init sock: %w", err)
	}

	if _, err := initConn.Write([]byte("ready")); err != nil {
		return fmt.Errorf("write 'ready' msg to init sock: %w", err)
	}
	// close immediately, to be 100% sure it doesn't leak into the container
	initConn.Close()

	return nil
}

func waitStart(id string) error {
	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, id, containerSockFilename),
	)
	if err != nil {
		return fmt.Errorf("listen on container sock: %w", err)
	}

	containerConn, err := listener.Accept()
	if err != nil {
		logrus.Errorf("failed to accept on container socket: %s", err)
		return fmt.Errorf("accept on container sock: %w", err)
	}

	b := make([]byte, 128)
	n, err := containerConn.Read(b)
	if err != nil {
		return fmt.Errorf("read bytes from container sock: %w", err)
	}

	msg := string(b[:n])
	if msg != "start" {
		return fmt.Errorf("expecting 'start' but received '%s'", msg)
	}

	// close immediately so we're sure no potential for leakage
	containerConn.Close()
	listener.Close()

	return nil
}

func execUserProcess(spec *specs.Spec) error {
	if err := os.Chdir(spec.Process.Cwd); err != nil {
		return fmt.Errorf("set working directory: %w", err)
	}

	bin, err := exec.LookPath(spec.Process.Args[0])
	if err != nil {
		return fmt.Errorf("find path of user process binary: %w", err)
	}

	args := spec.Process.Args
	env := os.Environ()

	if err := syscall.Exec(bin, args, env); err != nil {
		return fmt.Errorf("execve (%s, %s, %v): %w", bin, args, env, err)
	}

	return nil
}
