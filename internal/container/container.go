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

	"github.com/sirupsen/logrus"

	"github.com/nixpig/anocir/internal/anosys"
	"github.com/nixpig/anocir/internal/hooks"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

const (
	containerRootDir      = "/var/lib/anocir/containers"
	initSockFilename      = "init.sock"
	containerSockFilename = "container.sock"
)

type Container struct {
	State           *specs.State
	Spec            *specs.Spec
	ConsoleSocket   string
	ConsoleSocketFD *int
	PIDFile         string
	Opts            *NewContainerOpts
}

type NewContainerOpts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
	Stdin         *os.File
	Stdout        *os.File
	Stderr        *os.File
}

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
			return fmt.Errorf("write container PID to file (%s): %w", c.PIDFile, err)
		}
	}

	return nil
}

func (c *Container) Init() error {
	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.CreateRuntime, c.State,
		); err != nil {
			return fmt.Errorf("exec createruntime hooks: %w", err)
		}
	}

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.CreateContainer, c.State,
		); err != nil {
			return fmt.Errorf("exec createcontainer hooks: %w", err)
		}
	}

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		c.ConsoleSocket != ""

	if useTerminal {
		logrus.Info("🥡 USING TERMINAL")
		logrus.Infof("console socket: %s", c.ConsoleSocket)
		var err error
		if c.ConsoleSocketFD, err = terminal.Setup(
			c.rootFS(),
			c.ConsoleSocket,
		); err != nil {
			return err
		}
		logrus.Infof("console socketfd: %d", *c.ConsoleSocketFD)
	}

	args := []string{"reexec"}

	logLevel := logrus.GetLevel()
	if logLevel == logrus.DebugLevel {
		args = append(args, "--debug")
	}

	csfd := strconv.Itoa(*c.ConsoleSocketFD)

	args = append(args, "--console-socket-fd", csfd)

	args = append(args, c.State.ID)

	cmd := exec.Command(
		"/proc/self/exe",
		args...,
	)

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, initSockFilename),
	)
	if err != nil {
		return fmt.Errorf("listen on init sock: %w", err)
	}
	defer listener.Close()

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := anosys.AdjustOOMScore(*c.Spec.Process.OOMScoreAdj); err != nil {
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
				if err := anosys.SetTimeOffsets(c.Spec.Linux.TimeOffsets); err != nil {
					return fmt.Errorf("set timens offsets: %w", err)
				}
			}
		}

		if ns.Path == "" {
			cloneFlags |= anosys.NamespaceTypeToFlag(ns.Type)
		} else {
			suffix := fmt.Sprintf(
				"/%s",
				anosys.NamespaceTypeToEnv(ns.Type),
			)
			if !strings.HasSuffix(ns.Path, suffix) &&
				ns.Type != specs.PIDNamespace {
				return fmt.Errorf("namespace type (%s) and path (%s) do not match", ns.Type, ns.Path)
			}

			if ns.Type == specs.MountNamespace {
				// mount namespaces do not work across threads, so this needs to be done
				// in single-threaded context in C before the reexec
				gonsEnv := fmt.Sprintf(
					"gons_%s=%s",
					anosys.NamespaceTypeToEnv(ns.Type),
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

	cmd.Stdin = c.Opts.Stdin
	cmd.Stdout = c.Opts.Stdout
	cmd.Stderr = c.Opts.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reexec container process: %w", err)
	}

	c.State.Pid = cmd.Process.Pid
	if err := c.Save(); err != nil {
		return fmt.Errorf("save container pid state: %w", err)
	}

	if c.Spec.Linux.Resources != nil {
		if anosys.IsUnifiedCGroupsMode() {
			logrus.Info("using cgroupsv2")
			if err := anosys.AddV2CGroups(
				c.State.ID,
				c.Spec.Linux.Resources,
				c.State.Pid,
			); err != nil {
				return err
			}
		} else if c.Spec.Linux.CgroupsPath != "" {
			logrus.Info("using cgroupsv1")
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

	logrus.Info("📦️ created")

	return nil
}

func (c *Container) Reexec() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var pty *terminal.Pty
	logrus.Infof("👹 create a new terminal: %d", *c.ConsoleSocketFD)
	if c.ConsoleSocketFD != nil {
		logrus.Infof("👹 create a new terminal: %d", *c.ConsoleSocketFD)
		var err error

		pty, err = terminal.NewPty()
		if err != nil {
			return fmt.Errorf("new pty: %w", err)
		}

		if c.Spec.Process.ConsoleSize != nil {
			unix.IoctlSetWinsize(int(pty.Slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
				Row: uint16(c.Spec.Process.ConsoleSize.Height),
				Col: uint16(c.Spec.Process.ConsoleSize.Width),
			})
		}

		if err := terminal.SendPty(
			*c.ConsoleSocketFD,
			pty,
		); err != nil {
			return fmt.Errorf("connect pty and socket: %w", err)
		}

		if err := pty.Connect(); err != nil {
			return fmt.Errorf("connect pty: %w", err)
		}
	}

	if err := anosys.MountRootfs(c.rootFS()); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	if err := anosys.MountProc(c.rootFS()); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	if err := anosys.MountSpecMounts(c.Spec.Mounts, c.rootFS()); err != nil {
		return fmt.Errorf("mount spec: %w", err)
	}

	if err := anosys.MountDefaultDevices(c.rootFS()); err != nil {
		return fmt.Errorf("mount default devices: %w", err)
	}

	if err := anosys.CreateDeviceNodes(c.Spec.Linux.Devices, c.rootFS()); err != nil {
		return fmt.Errorf("mount devices from spec: %w", err)
	}

	if err := anosys.CreateDefaultSymlinks(c.rootFS()); err != nil {
		return fmt.Errorf("create default symlinks: %w", err)
	}

	if c.ConsoleSocketFD != nil && c.Spec.Process.Terminal {

		target := filepath.Join(c.rootFS(), "dev/console")

		if _, err := os.Stat(target); os.IsNotExist(err) {
			f, err := os.Create(target)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("create device target if not exists: %w", err)
			}
			if f != nil {
				f.Close()
			}
		}

		f, err := os.Stat(target)
		if err != nil {
			logrus.Errorf("unable to stat %s: %s", target, err)
			return err
		}

		logrus.Infof("fileinfo: %+v", f)

		logrus.Infof("slave name: %s", pty.Slave.Name())

		if err := syscall.Mount(
			pty.Slave.Name(),
			target,
			"bind",
			syscall.MS_BIND,
			"",
		); err != nil {
			return fmt.Errorf("mount dev/console device: %w", err)
		}
	}

	// wait a sec for init sock to be ready before dialing
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(
			containerRootDir,
			c.State.ID,
			initSockFilename,
		)); errors.Is(err, os.ErrNotExist) {
			time.Sleep(time.Millisecond * 100)
			continue
		}
	}

	initConn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, initSockFilename),
	)
	if err != nil {
		return fmt.Errorf("dial init sock: %w", err)
	}

	if _, err := initConn.Write([]byte("ready")); err != nil {
		return fmt.Errorf("write 'ready' msg to init sock: %w", err)
	}
	// close immediately, rather than defering
	initConn.Close()

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, containerSockFilename),
	)
	if err != nil {
		return fmt.Errorf("listen on container sock: %w", err)
	}

	logrus.Info("🉑 accepting")
	containerConn, err := listener.Accept()
	if err != nil {
		logrus.Errorf("☠️ accept on container socket: %s", err)
		return fmt.Errorf("accept on container sock: %w", err)
	}

	logrus.Info("🥪 read...")
	b := make([]byte, 128)
	n, err := containerConn.Read(b)
	if err != nil {
		return fmt.Errorf("read bytes from container sock: %w", err)
	}

	logrus.Info("⛳️ waiting for start")
	msg := string(b[:n])
	if msg != "start" {
		return fmt.Errorf("expecting 'start' but received '%s'", msg)
	}
	logrus.Info("🏌️ got start")

	containerConn.Close()
	listener.Close()

	if c.Spec.Process == nil {
		return errors.New("process is required")
	}

	logrus.Debug("pivot root")
	if err := anosys.PivotRoot(c.rootFS()); err != nil {
		return err
	}
	logrus.Debug("pivoted root!")

	if c.Spec.Linux.Sysctl != nil {
		if err := anosys.SetSysctl(c.Spec.Linux.Sysctl); err != nil {
			return fmt.Errorf("set sysctl: %w", err)
		}
	}

	if err := anosys.MountMaskedPaths(
		c.Spec.Linux.MaskedPaths,
	); err != nil {
		return err
	}

	if err := anosys.MountReadonlyPaths(
		c.Spec.Linux.ReadonlyPaths,
	); err != nil {
		return err
	}

	if err := anosys.SetRootfsMountPropagation(
		c.Spec.Linux.RootfsPropagation,
	); err != nil {
		return err
	}

	if err := anosys.MountRootReadonly(
		c.Spec.Root.Readonly,
	); err != nil {
		return err
	}

	hasUTSNamespace := slices.ContainsFunc(
		c.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	)

	if hasUTSNamespace {
		if err := syscall.Sethostname([]byte(c.Spec.Hostname)); err != nil {
			return err
		}

		if err := syscall.Setdomainname([]byte(c.Spec.Domainname)); err != nil {
			return err
		}
	}

	if err := anosys.SetRlimits(c.Spec.Process.Rlimits); err != nil {
		return fmt.Errorf("set rlimits: %w", err)
	}

	if c.Spec.Process.Capabilities != nil {
		if err := anosys.SetCapabilities(c.Spec.Process.Capabilities); err != nil {
			return fmt.Errorf("set capabilities: %w", err)
		}
	}

	if c.Spec.Process.NoNewPrivileges {
		if err := anosys.SetNoNewPrivs(); err != nil {
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	if c.Spec.Process.Scheduler != nil {
		if err := anosys.SetSchedAttrs(c.Spec.Process.Scheduler); err != nil {
			return fmt.Errorf("set sched attrs: %w", err)
		}
	}

	if c.Spec.Process.IOPriority != nil {
		if err := anosys.SetIOPriority(c.Spec.Process.IOPriority); err != nil {
			return fmt.Errorf("set ioprio: %w", err)
		}
	}

	if err := anosys.SetUser(&c.Spec.Process.User); err != nil {
		return fmt.Errorf("set user: %w", err)
	}

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.StartContainer, c.State,
		); err != nil {
			return fmt.Errorf("exec startcontainer hooks: %w", err)
		}
	}

	if err := os.Chdir(c.Spec.Process.Cwd); err != nil {
		return fmt.Errorf("set working directory: %w", err)
	}

	bin, err := exec.LookPath(c.Spec.Process.Args[0])
	if err != nil {
		return fmt.Errorf("find path of user process binary: %w", err)
	}

	args := c.Spec.Process.Args
	env := os.Environ()

	if err := syscall.Exec(bin, args, env); err != nil {
		return fmt.Errorf("execve (%s, %s, %v): %w", bin, args, env, err)
	}

	panic("if you got here then something went horribly wrong")
}

func (c *Container) Start() error {
	logrus.Info("🚀 START")
	if c.Spec.Process == nil {
		c.State.Status = specs.StateStopped
		if err := c.Save(); err != nil {
			return err
		}
		// nothing to do; silent return
		return nil
	}

	if !c.canBeStarted() {
		return fmt.Errorf("container cannot be started in current state (%s)", c.State.Status)
	}

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker
			c.Spec.Hooks.Prestart, c.State,
		); err != nil {
			logrus.Errorf("failed to exec prestart hooks: %s", err)
			return fmt.Errorf("execute prestart hooks: %w", err)
		}
	}

	logrus.Info("💬 dial the container socket")
	conn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, containerSockFilename),
	)
	if err != nil {
		logrus.Errorf("failed to dial container sock: %s", err)
		return fmt.Errorf("dial container sock: %w", err)
	}

	logrus.Info("✍️ write to container socke")
	if _, err := conn.Write([]byte("start")); err != nil {
		logrus.Errorf("failed to write start to sock: %s", err)
		return fmt.Errorf("write 'start' msg to container sock: %w", err)
	}
	conn.Close()

	c.State.Status = specs.StateRunning
	if err := c.Save(); err != nil {
		return fmt.Errorf("save state running: %w", err)
	}

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.Poststart, c.State,
		); err != nil {
			return fmt.Errorf("exec poststart hooks: %w", err)
		}
	}

	return nil
}

func (c *Container) Delete(force bool) error {
	if !force && !c.canBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state (%s) try using '--force'", c.State.Status)
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

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.Poststop, c.State,
		); err != nil {
			fmt.Printf("Warning: failed to exec poststop hooks: %s\n", err)
		}
	}

	return nil
}

func (c *Container) Kill(sig string) error {
	if !c.canBeKilled() {
		return fmt.Errorf("container cannot be killed in current state (%s)", c.State.Status)
	}

	if err := anosys.SendSignal(c.State.Pid, sig); err != nil {
		return fmt.Errorf("send signal '%s' to process '%d': %w", sig, c.State.Pid, err)
	}

	c.State.Status = specs.StateStopped
	if err := c.Save(); err != nil {
		return fmt.Errorf("save stopped state: %w", err)
	}

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.Poststop, c.State,
		); err != nil {
			fmt.Println("Warning: failed to execute poststop hooks")
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

func Exists(containerID string) bool {
	_, err := os.Stat(filepath.Join(containerRootDir, containerID))

	return err == nil
}
