package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/nixpig/brownie/container/filesystem"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Reexec1(log *zerolog.Logger) error {
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
	cmd := exec.Command(
		"/proc/self/exe",
		[]string{"reexec", "--stage", "2", c.ID()}...,
	)

	c.initIPC.ch <- []byte("ready")

	if err := ipc.WaitForMsg(listCh, "start", func() error {
		if err := cmd.Run(); err != nil {
			c.SetStatus(specs.StateStopped)
			if err := c.CSave(); err != nil {
				return fmt.Errorf("(start 1) write state file: %w", err)
			}

			return err
		}
		return nil
	}); err != nil {
		return err
	}

	c.SetStatus(specs.StateRunning)
	if err := c.CSave(); err != nil {
		// do something with err??
		log.Error().Err(err).Msg("⁉️ host save state running")
		fmt.Println(err)
		return err
	}

	if err := c.ExecHooks("poststart"); err != nil {
		// TODO: how to handle this (log a warning) from start command??
		// FIXME: needs to 'log a warning'
		fmt.Println("WARNING: ", err)
	}

	return nil
}
