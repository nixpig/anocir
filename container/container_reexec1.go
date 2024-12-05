package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/nixpig/brownie/container/filesystem"
	"github.com/nixpig/brownie/container/terminal"
	"github.com/nixpig/brownie/internal/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Reexec1(log *zerolog.Logger) error {
	var err error
	c.initIPC.ch, c.initIPC.closer, err = ipc.NewSender(filepath.Join("/var/lib/brownie/containers", c.ID(), initSockFilename))
	if err != nil {
		return fmt.Errorf("create init sock sender: %w", err)
	}
	defer c.initIPC.closer()

	// set up the socket _before_ pivot root
	if err := os.RemoveAll(
		filepath.Join("/var/lib/brownie/containers", c.ID(), containerSockFilename),
	); err != nil {
		return fmt.Errorf("remove socket before creating: %w", err)
	}

	listCh, listCloser, err := ipc.NewReceiver(filepath.Join("/var/lib/brownie/containers", c.ID(), containerSockFilename))
	if err != nil {
		return fmt.Errorf("create new socket receiver channel: %w", err)
	}
	defer listCloser()

	log.Info().Msg("before setup rootfs")
	if err := filesystem.SetupRootfs(c.Rootfs(), c.Spec); err != nil {
		return fmt.Errorf("setup rootfs: %w", err)
	}
	log.Info().Msg("after setup rootfs")

	if c.Spec.Process != nil && c.Spec.Process.OOMScoreAdj != nil {
		if err := os.WriteFile(
			"/proc/self/oom_score_adj",
			[]byte(strconv.Itoa(*c.Spec.Process.OOMScoreAdj)),
			0644,
		); err != nil {
			return fmt.Errorf("create oom score adj file: %w", err)
		}
	}

	if c.State.ConsoleSocket != nil {
		if err := terminal.SetupConsole(*c.State.ConsoleSocket); err != nil {
			log.Error().Err(err).Int("fd", *c.State.ConsoleSocket).Msg("failed to setup console")
			return fmt.Errorf("setup console: %w", err)
		}
	} else {
		// TODO: fall back to dup2 on stdin, stdout, stderr from c.Opts??
		fmt.Println("TODO: implement fallback stdio??")
	}

	cmd := exec.Command(
		"/proc/self/exe",
		[]string{"reexec", "--stage", "2", c.ID()}...,
	)

	// cmd.SysProcAttr.Unshareflags = cmd.SysProcAttr.Unshareflags | syscall.CLONE_NEWUSER
	// cmd.SysProcAttr.Cloneflags = cmd.SysProcAttr.Cloneflags | syscall.CLONE_NEWUSER

	log.Info().Msg("sending ready to channel")
	c.initIPC.ch <- []byte("ready")

	log.Info().Msg("waiting on start")
	if err := ipc.WaitForMsg(listCh, "start", func() error {
		log.Info().Msg("starting...")
		if err := cmd.Start(); err != nil {
			log.Error().Err(err).Msg("🔷 failed to start container")
			c.SetStatus(specs.StateStopped)
			if err := c.Save(); err != nil {
				return fmt.Errorf("(start 1) write state file: %w", err)
			}

			return err
		}

		c.SetStatus(specs.StateRunning)
		if err := c.Save(); err != nil {
			// do something with err??
			log.Error().Err(err).Msg("⁉️ host save state running")
			fmt.Println(err)
			return fmt.Errorf("save host container state: %w", err)
		}

		// FIXME: do these need to move up before the cmd.Wait call??
		if err := c.ExecHooks("poststart", log); err != nil {
			// TODO: how to handle this (log a warning) from start command??
			// FIXME: needs to 'log a warning'
			log.Warn().Err(err).Msg("failed to execute poststart hook")
			fmt.Println("WARNING: ", err)
		}

		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("ERROR IN WAITING IN REEXEC1")
			return fmt.Errorf("waiting for cmd wait in reexec: %w", err)
		}

		return nil
	}); err != nil {
		log.Error().Err(err).Msg("error in waitformsg")
		return err
	}

	return nil
}
