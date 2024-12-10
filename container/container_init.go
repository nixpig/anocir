package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/namespace"
	"github.com/nixpig/brownie/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

func (c *Container) Init(reexec string, arg string, log *zerolog.Logger) error {
	if err := c.ExecHooks("createRuntime", log); err != nil {
		return fmt.Errorf("execute createruntime hooks: %w", err)
	}

	if err := c.ExecHooks("createContainer", log); err != nil {
		return fmt.Errorf("execute createcontainer hooks: %w", err)
	}

	initSockAddr := filepath.Join("/var/lib/brownie/containers", c.ID(), initSockFilename)

	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewReceiver(initSockAddr)
	if err != nil {
		return fmt.Errorf("create init ipc receiver: %w", err)
	}
	defer c.initIPC.closer()

	reexecCmd := exec.Command(
		reexec,
		[]string{arg, "--stage", "1", c.ID()}...,
	)

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		c.Opts.ConsoleSocket != ""

	if useTerminal {
		prev, err := os.Getwd()
		if err != nil {
			log.Error().Err(err).Msg("failed to get current working directory")
			return fmt.Errorf("get cwd: %w", err)
		}

		if err := os.Chdir(c.Rootfs()); err != nil {
			log.Error().Err(err).Msg("failed to change to container root dir")
			return fmt.Errorf("change to container root dir: %w", err)
		}

		cwd, err := os.Getwd()
		if err != nil {
			log.Error().Err(err).Msg("failed to get cwd")
			return fmt.Errorf("get cwd: %w", err)
		}
		log.Info().Str("cwd", cwd).Msg("INIT current working directory")

		if err := os.Symlink(c.Opts.ConsoleSocket, "./console-socket"); err != nil {
			log.Error().Err(err).Msg("failed to symlink console socket")
			return fmt.Errorf("symlink console socket: %w", err)
		}

		consoleSocket, err := terminal.NewPtySocket(
			"./console-socket",
		)
		if err != nil {
			return fmt.Errorf("create terminal socket: %w", err)
		}
		c.State.ConsoleSocket = &consoleSocket.SocketFd

		if err := os.Chdir(prev); err != nil {
			log.Error().Err(err).Msg("failed to change back to previous directory")
			return fmt.Errorf("change back to prev dir: %w", err)
		}
	}

	if c.Spec.Linux.CgroupsPath != "" && c.Spec.Linux.Resources != nil {
		staticPath := cgroup1.StaticPath(c.Spec.Linux.CgroupsPath)

		cg, err := cgroup1.New(
			staticPath,
			&specs.LinuxResources{
				Devices: c.Spec.Linux.Resources.Devices,
			},
		)
		if err != nil {
			return fmt.Errorf("apply cgroups (path: %s): %w", c.Spec.Linux.CgroupsPath, err)
		}
		defer cg.Delete()

		cg.Add(cgroup1.Process{Pid: c.PID()})
	}

	cloneFlags := uintptr(0)
	unshareFlags := uintptr(0)

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

		ns := namespace.LinuxNamespace(ns)
		flag, err := ns.ToFlag()
		if err != nil {
			return fmt.Errorf("convert namespace to flag: %w", err)
		}

		if ns.Path == "" {
			cloneFlags |= flag
		} else {

			if !strings.HasSuffix(ns.Path, fmt.Sprintf("/%s", ns.ToEnv())) &&
				ns.Type != specs.PIDNamespace {
				return fmt.Errorf("namespace type (%s) and path (%s) do not match", ns.Type, ns.Path)
			}

			// TODO: align so the same mechanism is used for all namespaces?
			if ns.Type == specs.MountNamespace {
				reexecCmd.Env = append(reexecCmd.Env, fmt.Sprintf("gons_mnt=%s", ns.Path))
			} else {
				fd, err := syscall.Open(ns.Path, syscall.O_RDONLY, 0666)
				if err != nil {
					log.Error().Err(err).Str("path", ns.Path).Str("type", string(ns.Type)).Msg("failed to open namespace path")
					return fmt.Errorf("open ns path: %w", err)
				}
				defer syscall.Close(fd)

				_, _, errno := syscall.RawSyscall(unix.SYS_SETNS, uintptr(fd), 0, 0)
				if errno != 0 {
					log.Error().Str("path", ns.Path).Int("errno", int(errno)).Msg("FAIELD THE RAWSYSCALL")
					return fmt.Errorf("errno: %w", err)
				}
			}
		}
	}

	reexecCmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   cloneFlags,
		Unshareflags: unshareFlags,
		UidMappings:  uidMappings,
		GidMappings:  gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		reexecCmd.Env = append(reexecCmd.Env, c.Spec.Process.Env...)
	}

	reexecCmd.Stdin = c.Opts.Stdin
	reexecCmd.Stdout = c.Opts.Stdout
	reexecCmd.Stderr = c.Opts.Stderr

	if err := reexecCmd.Start(); err != nil {
		return fmt.Errorf("start reexec container: %w", err)
	}

	pid := reexecCmd.Process.Pid
	c.SetPID(pid)
	if err := c.Save(); err != nil {
		return fmt.Errorf("save pid for reexec: %w", err)
	}

	if c.Opts.PIDFile != "" {
		if err := os.WriteFile(
			c.Opts.PIDFile,
			[]byte(strconv.Itoa(pid)),
			0666,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", c.Opts.PIDFile, err)
		}
	}

	if err := reexecCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach reexec container: %w", err)
	}

	return ipc.WaitForMsg(c.initIPC.ch, "ready", func() error {
		c.SetStatus(specs.StateCreated)
		if err := c.Save(); err != nil {
			return fmt.Errorf("save created state: %w", err)
		}
		return nil
	})
}
