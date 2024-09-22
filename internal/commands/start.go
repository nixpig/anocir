package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/nixpig/brownie/internal"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts, log *zerolog.Logger) error {
	state, err := internal.GetState(opts.ID)
	if err != nil {
		log.Error().Err(err).Msg("start: get state")
		return fmt.Errorf("get state: %w", err)
	}

	if state.Status != specs.StateCreated {
		log.Error().Err(err).Msg("start: check state")
		return errors.New("container not created")
	}

	configJSON, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		log.Error().Err(err).Msg("start: read config")
		return fmt.Errorf("read config file: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		log.Error().Err(err).Msg("unmarshal config")
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	// 7. Invoke startContainer hooks
	if spec.Hooks != nil {
		if err := internal.ExecHooks(spec.Hooks.StartContainer); err != nil {
			log.Error().Err(err).Msg("start: exec startcontainer hooks")
			return fmt.Errorf("execute startContainer hooks: %w", err)
		}
	}

	// 8. TODO: Run the user-specified program from 'process' in the container
	// and update state to Running
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)
	containerSockAddr := filepath.Join(containerPath, "container.sock")
	conn, err := net.Dial("unix", containerSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("start: dial socket")
		return fmt.Errorf("dial socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("start: send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}

	state.Status = specs.StateRunning
	if err := internal.SaveState(state); err != nil {
		log.Error().Err(err).Msg("failed to save state")
	}

	b, err := io.ReadAll(conn)
	if err != nil {
		log.Error().Err(err).Msg("start: read response")
		return fmt.Errorf("reading response from socket: %w", err)
	}

	// FIXME: how do we redirect this to the stdout of the calling process?
	// E.g. when being run in tests.
	log.Info().Str("output", string(b)).Msg("run command output")
	fmt.Fprint(os.Stdout, string(b)) // this doesn't work via tests :/
	fmt.Println("FOO")

	// 9. Invoke poststart hooks
	if spec.Hooks != nil {
		if err := internal.ExecHooks(spec.Hooks.Poststart); err != nil {
			log.Error().Err(err).Msg("start: exec poststart hooks")
			return fmt.Errorf("execute poststart hooks: %w", err)
		}
	}

	// FIXME: spec is unclear what this should be
	// the tests are expecting it to be stopped, I think :/
	state.Status = specs.StateStopped
	if err := internal.SaveState(state); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}
