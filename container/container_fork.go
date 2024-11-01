package container

import (
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
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func (c *Container) Fork() error {
	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(filepath.Join(c.Bundle(), initSockFilename))
	if err != nil {
		return err
	}
	defer c.initIPC.closer()

	// if opts.ConsoleSocketFD != 0 {
	// 	log.Info().Msg("creating new terminal pty")
	// 	pty, err := terminal.NewPty()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer pty.Close()
	//
	// 	log.Info().Msg("connecting to terminal pty")
	// 	if err := pty.Connect(); err != nil {
	// 		return err
	// 	}
	//
	// 	log.Info().Msg("opening terminal pty socket")
	// 	consoleSocketPty := terminal.OpenPtySocket(
	// 		opts.ConsoleSocketFD,
	// 		opts.ConsoleSocketPath,
	// 	)
	// 	defer consoleSocketPty.Close()
	//
	// 	// FIXME: how do we pass ptysocket struct between fork?
	// 	log.Info().Msg("send message over terminal pty socket")
	// 	if err := consoleSocketPty.SendMsg(pty); err != nil {
	// 		return err
	// 	}
	// }

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join(c.Bundle(), containerSockFilename),
	); err != nil {
		return err
	}

	listCh, listCloser, err := ipc.NewReceiver(filepath.Join(c.Bundle(), containerSockFilename))
	if err != nil {
		return err
	}
	defer listCloser()

	if err := filesystem.SetupRootfs(c.Rootfs(), c.Spec); err != nil {
		return err
	}

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := os.WriteFile(
			"/proc/self/oom_score_adj",
			[]byte(strconv.Itoa(*c.Spec.Process.OOMScoreAdj)),
			0644,
		); err != nil {
			return err
		}
	}

	c.initIPC.ch <- []byte("ready")

	startErr := ipc.WaitForMsg(listCh, "start", func() error {
		if err := filesystem.PivotRoot(c.Rootfs()); err != nil {
			return err
		}

		if err := filesystem.MountMaskedPaths(
			c.Spec.Linux.MaskedPaths,
		); err != nil {
			return err
		}

		if err := filesystem.MountReadonlyPaths(
			c.Spec.Linux.ReadonlyPaths,
		); err != nil {
			return err
		}

		if c.Spec.Linux.RootfsPropagation != "" {
			if err := syscall.Mount("", "/", "", filesystem.MountOptions[c.Spec.Linux.RootfsPropagation].Flag, ""); err != nil {
				return err
			}
		}

		if c.Spec.Root.Readonly {
			// FIXME: subsequent attempts to update container state fail, either by
			// write to state.json (readonly filesystem) or write to db (readonly db)
			// probably we need to send message to a socket that handles it?
			if err := syscall.Mount("", "/", "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
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
				c.SetStatus(specs.StateStopped)
				if err := c.CSave(); err != nil {
					return fmt.Errorf("write state file: %w", err)
				}
				return err
			}

			if err := syscall.Setdomainname(
				[]byte(c.Spec.Domainname),
			); err != nil {
				c.SetStatus(specs.StateStopped)
				if err := c.CSave(); err != nil {
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
					c.SetStatus(specs.StateStopped)
					if err := c.CSave(); err != nil {
						return fmt.Errorf("write state file: %w", err)
					}
					return err
				}
			}

			if c.Spec.Process.Rlimits != nil {
				if err := cgroups.SetRlimits(c.Spec.Process.Rlimits); err != nil {
					c.SetStatus(specs.StateStopped)
					if err := c.CSave(); err != nil {
						return fmt.Errorf("write state file: %w", err)
					}
					return err
				}
			}
		}

		c.SetStatus(specs.StateRunning)
		if err := c.CSave(); err != nil {
			// do something with err??
			fmt.Println(err)
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

		cmd.Run()

		c.SetStatus(specs.StateStopped)
		if err := c.CSave(); err != nil {
			return fmt.Errorf("write state file: %w", err)
		}

		return nil
	})

	return startErr
}
