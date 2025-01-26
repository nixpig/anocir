package container

import (
	"bytes"
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

	"github.com/nixpig/anocir/internal/capabilities"
	"github.com/nixpig/anocir/internal/cgroups"
	"github.com/nixpig/anocir/internal/filesystem"
	"github.com/nixpig/anocir/internal/hooks"
	"github.com/nixpig/anocir/internal/iopriority"
	"github.com/nixpig/anocir/internal/scheduler"
	"github.com/nixpig/anocir/internal/specconv"
	"github.com/nixpig/anocir/internal/sysctl"
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
}

type NewContainerOpts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
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

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := os.WriteFile(
			"/proc/self/oom_score_adj",
			[]byte(strconv.Itoa(*c.Spec.Process.OOMScoreAdj)),
			0644,
		); err != nil {
			return fmt.Errorf("create oom score adj file: %w", err)
		}
	}

	cmd := exec.Command("/proc/self/exe", "reexec", c.State.ID)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, initSockFilename),
	)
	if err != nil {
		return fmt.Errorf("listen on init sock: %w", err)
	}
	defer listener.Close()

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.CreateContainer, c.State,
		); err != nil {
			return fmt.Errorf("exec createcontainer hooks: %w", err)
		}
	}

	cloneFlags := uintptr(0)

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap

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
				var tos bytes.Buffer

				for clock, offset := range c.Spec.Linux.TimeOffsets {
					if n, err := tos.WriteString(
						fmt.Sprintf("%s %d %d\n", clock, offset.Secs, offset.Nanosecs),
					); err != nil || n == 0 {
						return fmt.Errorf("write time offsets")
					}
				}

				if err := os.WriteFile(
					"/proc/self/timens_offsets",
					tos.Bytes(),
					0644,
				); err != nil {
					return fmt.Errorf("write timens offsets: %w", err)
				}
			}
		}

		if ns.Path == "" {
			cloneFlags |= specconv.NamespaceTypeToFlag(ns.Type)
		} else {
			suffix := fmt.Sprintf(
				"/%s",
				specconv.NamespaceTypeToEnv(ns.Type),
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
					specconv.NamespaceTypeToEnv(ns.Type),
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

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  cloneFlags,
		UidMappings: uidMappings,
		GidMappings: gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		cmd.Env = append(cmd.Env, c.Spec.Process.Env...)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reexec container process: %w", err)
	}

	c.State.Pid = cmd.Process.Pid
	if err := c.Save(); err != nil {
		return fmt.Errorf("save container pid state: %w", err)
	}

	if c.Spec.Linux.CgroupsPath != "" && c.Spec.Linux.Resources != nil {
		if cgroups.IsUnified() {
			if err := cgroups.AddV2(
				c.State.ID,
				c.Spec.Linux.Resources.Devices,
				c.State.Pid,
			); err != nil {
				return err
			}
		} else {
			if err := cgroups.AddV1(
				c.Spec.Linux.CgroupsPath,
				c.Spec.Linux.Resources.Devices,
				c.State.Pid,
			); err != nil {
				return err
			}
		}
	}

	if err := cmd.Process.Release(); err != nil {
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

func (c *Container) Reexec() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var pty *terminal.Pty
	if c.ConsoleSocketFD != nil {
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

	if err := filesystem.SetupRootfs(c.rootFS(), c.Spec); err != nil {
		return fmt.Errorf("setup rootfs: %w", err)
	}

	if c.ConsoleSocketFD != nil && c.Spec.Process.Terminal {
		dev := filesystem.Device{
			Source: pty.Slave.Name(),
			Target: filepath.Join(c.rootFS(), "dev/console"),
			Fstype: "bind",
			Flags:  syscall.MS_BIND,
			Data:   "",
		}
		if err := dev.Mount(); err != nil {
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

	containerConn, err := listener.Accept()
	if err != nil {
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

	containerConn.Close()
	listener.Close()

	if c.Spec.Process == nil {
		return errors.New("process is required")
	}

	logfile, err := os.OpenFile(filepath.Join(containerRootDir, c.State.ID, "log.txt"), 0755, os.FileMode(os.O_CREATE|os.O_RDWR))
	if err != nil {
		return err
	}
	defer logfile.Close()

	if err := filesystem.PivotRoot(c.rootFS()); err != nil {
		logfile.Write([]byte(err.Error()))
		return err
	}

	if c.Spec.Linux.Sysctl != nil {
		if err := sysctl.SetSysctl(c.Spec.Linux.Sysctl); err != nil {
			logfile.Write([]byte(fmt.Sprintf("setctl: %s", err)))
			return fmt.Errorf("set sysctl: %w", err)
		}
	}

	if err := filesystem.MountMaskedPaths(
		c.Spec.Linux.MaskedPaths,
	); err != nil {
		logfile.Write([]byte(fmt.Sprintf("mount masked paths: %s", err)))
		return err
	}

	if err := filesystem.MountReadonlyPaths(
		c.Spec.Linux.ReadonlyPaths,
	); err != nil {
		logfile.Write([]byte(fmt.Sprintf("mount readonly paths: %s", err)))
		return err
	}

	if err := filesystem.SetRootfsMountPropagation(
		c.Spec.Linux.RootfsPropagation,
	); err != nil {
		logfile.Write([]byte(fmt.Sprintf("set rootfs mount propagation: %s", err)))
		return err
	}

	if err := filesystem.MountRootReadonly(
		c.Spec.Root.Readonly,
	); err != nil {
		logfile.Write([]byte(fmt.Sprintf("mount root readonly: %s", err)))
		return err
	}

	logfile.Write([]byte("after mounts"))

	if slices.ContainsFunc(
		c.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname([]byte(c.Spec.Hostname)); err != nil {
			logfile.Write([]byte(fmt.Sprintf("set hostname: %s", err)))
			return err
		}

		if err := syscall.Setdomainname([]byte(c.Spec.Domainname)); err != nil {
			logfile.Write([]byte(fmt.Sprintf("set domainname: %s", err)))
			return err
		}
	}

	logfile.Write([]byte("after namespaces"))

	if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
		logfile.Write([]byte(fmt.Sprintf("set rlimits: %s", err)))
		return err
	}

	logfile.Write([]byte("after rlimits"))

	if err := capabilities.SetCapabilities(
		c.Spec.Process.Capabilities,
	); err != nil {
		logfile.Write([]byte(fmt.Sprintf("set capabilities: %s", err)))
		return err
	}

	logfile.Write([]byte("after caps"))

	if c.Spec.Process.NoNewPrivileges {
		if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
			logfile.Write([]byte(fmt.Sprintf("set no new privileges: %s", err)))
			return fmt.Errorf("set no new privileges: %w", err)
		}
	}

	logfile.Write([]byte("after privs"))

	if c.Spec.Process.Scheduler != nil {
		policy, err := scheduler.PolicyToInt(c.Spec.Process.Scheduler.Policy)
		if err != nil {
			logfile.Write([]byte(fmt.Sprintf("get scheduler policy flags: %s", err)))
			return fmt.Errorf("scheduler policy to int: %w", err)
		}

		flags, err := scheduler.FlagsToInt(c.Spec.Process.Scheduler.Flags)
		if err != nil {
			logfile.Write([]byte(fmt.Sprintf("convert scheduler flags to int: %s", err)))
			return fmt.Errorf("scheduler flags to int: %w", err)
		}

		schedAttr := unix.SchedAttr{
			Deadline: c.Spec.Process.Scheduler.Deadline,
			Flags:    uint64(flags),
			Size:     unix.SizeofSchedAttr,
			Nice:     c.Spec.Process.Scheduler.Nice,
			Period:   c.Spec.Process.Scheduler.Period,
			Policy:   uint32(policy),
			Priority: uint32(c.Spec.Process.Scheduler.Priority),
			Runtime:  c.Spec.Process.Scheduler.Runtime,
		}

		if err := unix.SchedSetAttr(0, &schedAttr, 0); err != nil {
			logfile.Write([]byte(fmt.Sprintf("set scheduler policy: %s", err)))
			return fmt.Errorf("set schedattrs: %w", err)
		}
	}

	logfile.Write([]byte("after sched"))

	if c.Spec.Process.IOPriority != nil {
		ioprio, err := iopriority.ToInt(c.Spec.Process.IOPriority)
		if err != nil {
			logfile.Write([]byte(fmt.Sprintf("iopriority to int: %s", err)))
			return fmt.Errorf("iopriority to int: %w", err)
		}

		if err := iopriority.SetIOPriority(ioprio); err != nil {
			logfile.Write([]byte(fmt.Sprintf("set ioprio: %s", err)))
			return fmt.Errorf("set iop: %w", err)
		}
	}

	logfile.Write([]byte("after ioprio"))

	if err := syscall.Setuid(int(c.Spec.Process.User.UID)); err != nil {
		logfile.Write([]byte(fmt.Sprintf("set uid: %s", err)))
		return fmt.Errorf("set UID: %w", err)
	}

	logfile.Write([]byte("after uid"))

	if err := syscall.Setgid(int(c.Spec.Process.User.GID)); err != nil {
		logfile.Write([]byte(fmt.Sprintf("set gid: %s", err)))
		return fmt.Errorf("set GID: %w", err)
	}

	logfile.Write([]byte("after gid"))

	additionalGids := make([]int, len(c.Spec.Process.User.AdditionalGids))
	for i, gid := range c.Spec.Process.User.AdditionalGids {
		additionalGids[i] = int(gid)
	}

	// FIXME: run 'linux_uid_mappings' and it causes it to 'disappear'
	// if err := syscall.Setgroups(additionalGids); err != nil {
	// 	logfile.Write([]byte(fmt.Sprintf("set additional gids: %s", err)))
	// 	return fmt.Errorf("set additional GIDs: %w", err)
	// }

	logfile.Write([]byte("after additional gids"))

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			c.Spec.Hooks.StartContainer, c.State,
		); err != nil {
			return fmt.Errorf("exec startcontainer hooks: %w", err)
		}
	}

	if err := os.Chdir(c.Spec.Process.Cwd); err != nil {
		logfile.Write([]byte(fmt.Sprintf("set working directory: %s", err)))
		return fmt.Errorf("set working directory: %w", err)
	}

	bin, err := exec.LookPath(c.Spec.Process.Args[0])
	if err != nil {
		logfile.Write([]byte(fmt.Sprintf("find path of process binary: %s", err)))
		return fmt.Errorf("find path of user process binary: %w", err)
	}

	args := c.Spec.Process.Args
	env := os.Environ()

	logfile.Write([]byte("exec the process :)"))
	if err := syscall.Exec(bin, args, env); err != nil {
		logfile.Write([]byte(fmt.Sprintf("exec process binary: %s", err)))
		return fmt.Errorf("execve (%s, %s, %v): %w", bin, args, env, err)
	}

	logfile.Write([]byte("time to panic!!"))

	panic("if you got here then something went horribly wrong")
}

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
		return fmt.Errorf("container cannot be started in current state (%s)", c.State.Status)
	}

	if c.Spec.Hooks != nil {
		if err := hooks.ExecHooks(
			//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker
			c.Spec.Hooks.Prestart, c.State,
		); err != nil {
			return fmt.Errorf("execute prestart hooks: %w", err)
		}
	}

	conn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, containerSockFilename),
	)
	if err != nil {
		return fmt.Errorf("dial container sock: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
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

func (c *Container) Kill(sig unix.Signal) error {
	if !c.canBeKilled() {
		return fmt.Errorf("container cannot be killed in current state (%s)", c.State.Status)
	}

	if err := syscall.Kill(c.State.Pid, sig); err != nil {
		return fmt.Errorf("send signal '%d' to process '%d': %w", sig, c.State.Pid, err)
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
