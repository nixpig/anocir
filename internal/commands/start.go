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
)

func Start(containerID string) error {
	state, err := internal.GetState(containerID)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	if state.Status != specs.StateCreated {
		return errors.New("container not created")
	}

	configJSON, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	// 7. Invoke startContainer hooks
	if spec.Hooks != nil {
		if err := internal.ExecHooks(spec.Hooks.StartContainer); err != nil {
			return fmt.Errorf("execute startContainer hooks: %w", err)
		}
	}

	// 8. TODO: Run the user-specified program from 'process' in the container
	// and update state to Running
	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", containerID)
	containerSockAddr := filepath.Join(containerPath, "container.sock")
	conn, err := net.Dial("unix", containerSockAddr)
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send start over ipc: %w", err)
	}

	b, err := io.ReadAll(conn)
	if err != nil {
		return fmt.Errorf("reading response from socket: %w", err)
	}

	// FIXME: print output from command run inside container??
	// presumably this needs to be redirected to the pty (if specified in config)?
	fmt.Println(string(b))

	// 9. Invoke poststart hooks
	if spec.Hooks != nil {
		if err := internal.ExecHooks(spec.Hooks.Poststart); err != nil {
			return fmt.Errorf("execute poststart hooks: %w", err)
		}
	}

	return nil
}
