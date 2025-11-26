package container

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"syscall"

	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/nixpig/anocir/internal/platform"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// envInitSockFD is the name of the environment variable used to pass the
// init sock file descriptor.
const envInitSockFD = "_ANOCIR_INIT_SOCK_FD"

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
		return fmt.Errorf("release container process: %w", err)
	}

	c.State.Status = specs.StateCreated
	if err := c.Save(); err != nil {
		return fmt.Errorf("save created state: %w", err)
	}

	return nil
}

func (c *Container) useTerminal() bool {
	return c.spec.Process != nil &&
		c.spec.Process.Terminal &&
		c.ConsoleSocket != ""
}
