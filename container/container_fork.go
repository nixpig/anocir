package container

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"

	"github.com/nixpig/brownie/container/capabilities"
	"github.com/nixpig/brownie/container/cgroups"
	"github.com/nixpig/brownie/container/filesystem"
	"github.com/nixpig/brownie/container/terminal"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type ForkOpts struct {
	ID                string
	InitSockAddr      string
	ConsoleSocketFD   int
	ConsoleSocketPath string
}

func (c *Container) Fork(opts *ForkOpts, log *zerolog.Logger, db *sql.DB) error {
	var err error
	log.Info().Msg("creating new init sender")
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(opts.InitSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("failed creating ipc sender")
		return err
	}
	defer c.initIPC.closer()

	if opts.ConsoleSocketFD != 0 {
		log.Info().Msg("creating new terminal pty")
		pty, err := terminal.NewPty()
		if err != nil {
			return err
		}
		defer pty.Close()

		log.Info().Msg("connecting to terminal pty")
		if err := pty.Connect(); err != nil {
			return err
		}

		log.Info().Msg("opening terminal pty socket")
		consoleSocketPty := terminal.OpenPtySocket(
			opts.ConsoleSocketFD,
			opts.ConsoleSocketPath,
		)
		defer consoleSocketPty.Close()

		// FIXME: how do we pass ptysocket struct between fork?
		log.Info().Msg("send message over terminal pty socket")
		if err := consoleSocketPty.SendMsg(pty); err != nil {
			return err
		}
	}

	// set up the socket _before_ pivot root
	log.Info().Msg("remove existing container socket")
	if err := os.RemoveAll(
		filepath.Join(c.State.Bundle, containerSockFilename),
	); err != nil {
		return err
	}

	log.Info().Msg("create new container socket receiver")
	listCh, listCloser, err := ipc.NewReceiver(filepath.Join(c.State.Bundle, containerSockFilename))
	if err != nil {
		log.Error().Err(err).Msg("failed to create new ipc receiver")
		return err
	}
	defer listCloser()

	log.Info().Msg("setup root filesystem")
	if err := filesystem.SetupRootfs(c.State.Bundle, c.Spec, log); err != nil {
		log.Error().Err(err).Msg("failed to setup rootfs")
		return err
	}

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := os.WriteFile(
			"/proc/self/oom_score_adj",
			[]byte(strconv.Itoa(*c.Spec.Process.OOMScoreAdj)),
			0644,
		); err != nil {
			log.Error().Err(err).Msg("failed to write oom_score_adj")
			return err
		}
	}

	log.Info().Msg("sending 'ready' msg")
	c.initIPC.ch <- []byte("ready")

	log.Info().Msg("waiting for 'start' msg")
	startErr := ipc.WaitForMsg(listCh, "start", func() error {
		if err := filesystem.PivotRoot(c.State.Bundle); err != nil {
			log.Error().Err(err).Msg("failed to pivot root")
			return err
		}

		if c.Spec.Linux.RootfsPropagation != "" {
			if err := syscall.Mount("", "/", "", filesystem.MountOptions[c.Spec.Linux.RootfsPropagation].Flag, ""); err != nil {
				log.Error().Err(err).Msg("failed to apply rootfs propagation")
				return err
			}
		}

		if c.Spec.Root.Readonly {
			// FIXME: subsequent attempts to update container state fail, either by
			// write to state.json (readonly filesystem) or write to db (readonly db)
			// probably we need to send message to a socket that handles it?
			if err := syscall.Mount("", "/", "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
				log.Error().Err(err).Msg("failed to remount rootfs as readonly")
				return err
			}
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

		c.State.Status = specs.StateRunning
		if err := c.SaveState(); err != nil {
			log.Error().Err(err).Msg("failed to save state")
		}

		cmd := exec.Command(
			c.Spec.Process.Args[0],
			c.Spec.Process.Args[1:]...,
		)

		cmd.Dir = c.Spec.Process.Cwd

		// cmd.SysProcAttr.Credential = &syscall.Credential{
		// 	Uid:    c.Spec.Process.User.UID,
		// 	Gid:    c.Spec.Process.User.GID,
		// 	Groups: c.Spec.Process.User.AdditionalGids,
		// }

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Info().Msg("BEFORE THE COMMAND RUN")
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

	return startErr
}
