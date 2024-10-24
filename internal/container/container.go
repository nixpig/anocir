package container

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nixpig/brownie/internal/container/capabilities"
	"github.com/nixpig/brownie/internal/container/cgroups"
	"github.com/nixpig/brownie/internal/container/filesystem"
	"github.com/nixpig/brownie/internal/container/lifecycle"
	"github.com/nixpig/brownie/internal/container/namespace"
	"github.com/nixpig/brownie/internal/container/terminal"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

const initSockFilename = "init.sock"
const containerSockFilename = "container.sock"

type Container struct {
	State *State
	Spec  *specs.Spec

	forkCmd *exec.Cmd
	initIPC ipcCtrl
	db      *sql.DB
}

type State struct {
	Version     string
	ID          string
	Bundle      string
	Annotations map[string]string
	Status      specs.ContainerState
	PID         int
}

type InitOpts struct {
	PIDFile       string
	ConsoleSocket string
	Stdin         *os.File
	Stdout        *os.File
	Stderr        *os.File
}

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ConsoleSocketFD   int
	ConsoleSocketPath string
}

type ipcCtrl struct {
	ch     chan []byte
	closer func() error
}

func New(
	id string,
	bundle string,
	status specs.ContainerState,
	log *zerolog.Logger,
	db *sql.DB,
) (*Container, error) {
	_, err := db.Query(`select id_ from containers_ where id_ = $id`, sql.Named("id", id))
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf(
			"container already exists (%s): %w",
			id, err,
		)
	}

	b, err := os.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal spec")
		return nil, fmt.Errorf("parse container config: %w", err)
	}

	if spec.Linux == nil {
		return nil, errors.New("only linux containers are supported")
	}

	absBundlePath, err := filepath.Abs(bundle)
	if err != nil {
		return nil, fmt.Errorf("construct absolute bundle path: %w", err)
	}

	state := &State{
		Version:     pkg.OCIVersion,
		ID:          id,
		Bundle:      absBundlePath,
		Annotations: map[string]string{},
		Status:      status,
	}

	// TODO: save to database
	query := `insert into containers_ (
		id_, version_, bundle_, pid_, status_, config_
	) values (
		$id, $version, $bundle, $pid, $status, $config
	)`

	if _, err := db.Exec(
		query,
		sql.Named("id", id),
		sql.Named("version", state.Version),
		sql.Named("bundle", state.Bundle),
		sql.Named("pid", state.PID),
		sql.Named("status", state.Status),
		sql.Named("config", string(b)),
	); err != nil {
		return nil, fmt.Errorf("insert into db: %w", err)
	}

	cntr := Container{
		State: state,
		Spec:  &spec,
		db:    db,
	}

	if err := cntr.Save(); err != nil {
		return nil, fmt.Errorf("save newly created container: %w", err)
	}

	return &cntr, nil
}

func (c *Container) Init(opts *InitOpts, log *zerolog.Logger) error {
	initSockAddr := filepath.Join(c.State.Bundle, initSockFilename)
	if err := os.RemoveAll(initSockAddr); err != nil {
		return fmt.Errorf("remove existing init socket: %w", err)
	}

	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewReceiver(initSockAddr)
	if err != nil {
		return fmt.Errorf("create init ipc receiver: %w", err)
	}
	defer c.initIPC.closer()

	useTerminal := c.Spec.Process != nil &&
		c.Spec.Process.Terminal &&
		opts.ConsoleSocket != ""

	var termFD int
	if useTerminal {
		termSock, err := terminal.New(opts.ConsoleSocket)
		if err != nil {
			return fmt.Errorf("create terminal socket: %w", err)
		}
		termFD = termSock.FD
	}

	c.forkCmd = exec.Command(
		"/proc/self/exe",
		[]string{
			"fork",
			c.State.ID,
			initSockAddr,
			strconv.Itoa(termFD),
		}...)

	var ambientCapsFlags []uintptr
	if c.Spec.Process != nil &&
		c.Spec.Process.Capabilities != nil {
		for _, cap := range c.Spec.Process.Capabilities.Ambient {
			ambientCapsFlags = append(
				ambientCapsFlags,
				uintptr(capabilities.Capabilities[cap]),
			)
		}
	}

	var cloneFlags uintptr
	if c.Spec.Linux.Namespaces != nil {
		for _, ns := range c.Spec.Linux.Namespaces {
			ns := namespace.LinuxNamespace(ns)
			flag, err := ns.ToFlag()
			if err != nil {
				return fmt.Errorf("convert namespace to flag: %w", err)
			}

			cloneFlags |= flag
		}
	}

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap

	// TODO: review if this is needed
	// if c.Spec.Process != nil {
	// cloneFlags |= syscall.CLONE_NEWUSER

	// uidMappings = append(uidMappings, syscall.SysProcIDMap{
	// 	ContainerID: int(c.Spec.Process.User.UID),
	// 	HostID:      os.Geteuid(),
	// 	Size:        1,
	// })
	//
	// gidMappings = append(gidMappings, syscall.SysProcIDMap{
	// 	ContainerID: int(c.Spec.Process.User.GID),
	// 	HostID:      os.Getegid(),
	// 	Size:        1,
	// })
	// }

	if c.Spec.Linux.UIDMappings != nil {
		for _, uidMapping := range c.Spec.Linux.UIDMappings {
			uidMappings = append(uidMappings, syscall.SysProcIDMap{
				ContainerID: int(uidMapping.ContainerID),
				HostID:      int(uidMapping.HostID),
				Size:        int(uidMapping.Size),
			})
		}
	}

	if c.Spec.Linux.GIDMappings != nil {
		for _, gidMapping := range c.Spec.Linux.GIDMappings {
			gidMappings = append(gidMappings, syscall.SysProcIDMap{
				ContainerID: int(gidMapping.ContainerID),
				HostID:      int(gidMapping.HostID),
				Size:        int(gidMapping.Size),
			})
		}
	}

	c.forkCmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps:                ambientCapsFlags,
		Cloneflags:                 cloneFlags,
		Unshareflags:               syscall.CLONE_NEWNS,
		GidMappingsEnableSetgroups: false,
		UidMappings:                uidMappings,
		GidMappings:                gidMappings,
	}

	if c.Spec.Process != nil && c.Spec.Process.Env != nil {
		c.forkCmd.Env = c.Spec.Process.Env
	}

	c.forkCmd.Stdin = opts.Stdin
	c.forkCmd.Stdout = opts.Stdout
	c.forkCmd.Stderr = opts.Stderr

	if err := c.forkCmd.Start(); err != nil {
		return fmt.Errorf("start fork container: %w", err)
	}

	pid := c.forkCmd.Process.Pid
	c.State.PID = pid
	if err := c.Save(); err != nil {
		return fmt.Errorf("save pid for fork: %w", err)
	}

	if err := c.forkCmd.Process.Release(); err != nil {
		return fmt.Errorf("detach fork container: %w", err)
	}

	if opts.PIDFile != "" {
		log.Info().Str("pidfile", opts.PIDFile).Int("pid", pid).Msg("writing pidfile")
		if err := os.WriteFile(
			opts.PIDFile,
			[]byte(strconv.Itoa(pid)),
			0666,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", opts.PIDFile, err)
		}
	}

	return ipc.WaitForMsg(c.initIPC.ch, "ready", func() error {
		c.State.Status = specs.StateCreated
		if err := c.Save(); err != nil {
			return fmt.Errorf("save created state: %w", err)
		}
		return nil
	})

	// p, err := os.FindProcess(pid)
	// if err != nil {
	// 	log.Error().Err(err).Int("pid", pid).Msg("failed to find process")
	// 	return -1, err
	// }

	// fmt.Println("waiting for process to exit", p.Pid)
	// o, err := p.Wait()
	// if err != nil {
	// 	log.Error().Err(err).Int("pid", pid).Msg("waiting for process to exit")
	// 	return -1, err
	// }
	//
	// fmt.Println(o)

}

func (c *Container) Fork(opts *ForkOpts, log *zerolog.Logger, db *sql.DB) error {
	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed creating ipc sender")
		return err
	}
	defer c.initIPC.closer()

	if opts.ConsoleSocketFD != 0 {
		pty, err := terminal.NewPty()
		if err != nil {
			return err
		}
		defer pty.Close()

		if err := pty.Connect(); err != nil {
			return err
		}

		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// FIXME: how do we pass ptysocket struct between fork?
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			return err
		}
	}

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join(c.State.Bundle, containerSockFilename),
	); err != nil {
		return err
	}

	listCh, listCloser, err := ipc.NewReceiver(filepath.Join(c.State.Bundle, containerSockFilename))
	if err != nil {
		log.Error().Err(err).Msg("failed to create new ipc receiver")
		return err
	}
	defer listCloser()

	if err := filesystem.SetupRootfs(c.State.Bundle, c.Spec); err != nil {
		log.Error().Err(err).Msg("failed to setup rootfs")
		return err
	}

	if slices.ContainsFunc(
		c.Spec.Linux.Namespaces,
		func(n specs.LinuxNamespace) bool {
			return n.Type == specs.UTSNamespace
		},
	) {
		if err := syscall.Sethostname(
			[]byte(c.Spec.Hostname),
		); err != nil {
			c.State.Status = specs.StateStopped
			if err := c.SaveState(); err != nil {
				log.Error().Err(err).Msg("failed to write state file")
				return fmt.Errorf("write state file: %w", err)
			}
			return err
		}

		if err := syscall.Setdomainname(
			[]byte(c.Spec.Domainname),
		); err != nil {
			c.State.Status = specs.StateStopped
			if err := c.SaveState(); err != nil {
				log.Error().Err(err).Msg("failed to write state file")
				return fmt.Errorf("write state file: %w", err)
			}
			return err
		}
	}

	if c.Spec.Process != nil {
		if c.Spec.Process.Capabilities != nil {
			if err := capabilities.SetCapabilities(
				c.Spec.Process.Capabilities,
			); err != nil {
				c.State.Status = specs.StateStopped
				if err := c.SaveState(); err != nil {
					log.Error().Err(err).Msg("failed to write state file")
					return fmt.Errorf("write state file: %w", err)
				}
				return err
			}
		}

		if c.Spec.Process.Rlimits != nil {
			if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
				c.State.Status = specs.StateStopped
				if err := c.SaveState(); err != nil {
					log.Error().Err(err).Msg("failed to write state file")
					return fmt.Errorf("write state file: %w", err)
				}
				return err
			}
		}
	}

	c.initIPC.ch <- []byte("ready")

	return ipc.WaitForMsg(listCh, "start", func() error {
		c.State.Status = specs.StateRunning
		if err := c.SaveState(); err != nil {
			log.Error().Err(err).Msg("failed to save state")
		}

		cmd := exec.Command(
			c.Spec.Process.Args[0],
			c.Spec.Process.Args[1:]...,
		)

		cmd.Dir = c.Spec.Process.Cwd

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Info().Msg("BEFORE THE  COMMAND RUN")
		cmd.Run()
		log.Info().Msg("AFTER THE COMMAND RUN")

		log.Info().Msg("UPDATING STATE FILE")
		c.State.Status = specs.StateStopped
		if err := c.SaveState(); err != nil {
			log.Error().Err(err).Msg("failed to write state file")
			return fmt.Errorf("write state file: %w", err)
		}
		log.Info().Msg("UPDATED STATE FILE")

		return nil
	})
}

func Load(id string, log *zerolog.Logger, db *sql.DB) (*Container, error) {
	state := State{}
	var c string

	row := db.QueryRow(`select id_, version_, bundle_, pid_, status_, config_ from containers_ where id_ = $id`, sql.Named("id", id))

	if err := row.Scan(
		&state.ID,
		&state.Version,
		&state.Bundle,
		&state.PID,
		&state.Status,
		&c,
	); err != nil {
		return nil, fmt.Errorf("scan container to struct: %w", err)
	}

	conf := specs.Spec{}
	if err := json.Unmarshal([]byte(c), &conf); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal state in loader")
		return nil, fmt.Errorf("unmarshall state to struct: %w", err)
	}

	cntr := &Container{
		State: &state,
		Spec:  &conf,
		db:    db,
	}

	if err := cntr.RefreshState(); err != nil {
		log.Error().Err(err).Msg("failed to refresh state")
		return nil, fmt.Errorf("refresh state: %w", err)
	}

	return cntr, nil
}

func (c *Container) RefreshState() error {
	b, err := os.ReadFile(filepath.Join(c.State.Bundle, "state.json"))
	if err != nil {
		fmt.Println("WARNING: unable to refresh from state file")
		return nil
	}

	if err := json.Unmarshal(b, c.State); err != nil {
		return fmt.Errorf("unmarshall refreshed state: %w", err)
	}

	return nil
}

func (c *Container) SaveState() error {
	b, err := json.Marshal(c.State)
	if err != nil {
		return err
	}
	if err := os.WriteFile("/state.json", b, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func (c *Container) Save() error {
	_, err := c.db.Exec(
		`update containers_ set 
		status_ = $status,
		pid_ = $pid,
		bundle_ = $bundle,
		version_ = $version
		where id_ = $id`,
		sql.Named("status", c.State.Status),
		sql.Named("id", c.State.ID),
		sql.Named("pid", c.State.PID),
		sql.Named("bundle", c.State.Bundle),
		sql.Named("version", c.State.Version),
	)
	if err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	b, err := json.Marshal(c.State)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(c.State.Bundle, "state.json"), b, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func (c *Container) Clean() error {
	return os.RemoveAll(c.State.Bundle)
}

func (c *Container) ExecHooks(hook string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var specHooks []specs.Hook
	switch hook {
	case "createRuntime":
		specHooks = c.Spec.Hooks.CreateRuntime
	case "createContainer":
		specHooks = c.Spec.Hooks.CreateContainer
	case "startContainer":
		specHooks = c.Spec.Hooks.StartContainer
	case "poststart":
		specHooks = c.Spec.Hooks.Poststart
	case "poststop":
		specHooks = c.Spec.Hooks.Poststop
	}

	return lifecycle.ExecHooks(specHooks)
}

func (c *Container) CanBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func (c *Container) CanBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateCreated
}

func (c *Container) CanBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}
