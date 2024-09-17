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
	log.Info().Str("id", opts.ID).Msg("starting container")
	state, err := internal.GetState(opts.ID)
	if err != nil {
		log.Error().Err(err).Msg("start: get state")
		return fmt.Errorf("get state: %w", err)
	}

	log.Info().Str("status", string(state.Status)).Msg("checking if created")
	if state.Status != specs.StateCreated {
		log.Error().Err(err).Msg("start: check state")
		return errors.New("container not created")
	}

	log.Info().Msg("reading config json")
	configJSON, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		log.Error().Err(err).Msg("start: read config")
		return fmt.Errorf("read config file: %w", err)
	}

	log.Info().Msg("unmarshaling config")
	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		log.Error().Err(err).Msg("unmarshal config")
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	// 7. Invoke startContainer hooks
	if spec.Hooks != nil {
		log.Info().Msg("executing startcontainer hooks")
		if err := internal.ExecHooks(spec.Hooks.StartContainer); err != nil {
			log.Error().Err(err).Msg("start: exec startcontainer hooks")
			return fmt.Errorf("execute startContainer hooks: %w", err)
		}
	}

	// 8. TODO: Run the user-specified program from 'process' in the container
	// and update state to Running
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", opts.ID)
	containerSockAddr := filepath.Join(containerPath, "container.sock")
	log.Info().Str("socket", containerSockAddr).Msg("dialing container socket")
	conn, err := net.Dial("unix", containerSockAddr)
	if err != nil {
		log.Error().Err(err).Msg("start: dial socket")
		return fmt.Errorf("dial socket: %w", err)
	}

	log.Info().Msg("sending start message")
	if _, err := conn.Write([]byte("start")); err != nil {
		log.Error().Err(err).Msg("start: send start message")
		return fmt.Errorf("send start over ipc: %w", err)
	}

	log.Info().Msg("reading from connection")
	b, err := io.ReadAll(conn)
	if err != nil {
		log.Error().Err(err).Msg("start: read response")
		return fmt.Errorf("reading response from socket: %w", err)
	}

	// FIXME: print output from command run inside container??
	// presumably this needs to be redirected to the pty (if specified in config)?
	log.Info().Str("output", string(b)).Msg("run command output")
	fmt.Fprint(os.Stdout, string(b))
	f, _ := os.OpenFile(
		"out.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	f.Write(b)
	conn.Write(b)

	// 9. Invoke poststart hooks
	if spec.Hooks != nil {
		log.Info().Msg("executing poststart hooks")
		if err := internal.ExecHooks(spec.Hooks.Poststart); err != nil {
			log.Error().Err(err).Msg("start: exec poststart hooks")
			return fmt.Errorf("execute poststart hooks: %w", err)
		}
	}

	return nil
}
