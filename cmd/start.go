package cmd

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
	"github.com/nixpig/brownie/pkg/config"
)

func Start(containerID string) error {
	containerPath := filepath.Join(BrownieRootDir, "containers", containerID)

	fc, err := os.ReadFile(filepath.Join(containerPath, "state.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("container not found")
		} else {
			return fmt.Errorf("stat container path: %w", err)
		}
	}

	var state pkg.State
	if err := json.Unmarshal(fc, &state); err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if state.Status != pkg.Created {
		return errors.New("container not created")
	}

	c, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(c, &cfg); err != nil {
		return fmt.Errorf("unmarshall config.json: %w", err)
	}

	// 7. Invoke startContainer hooks
	if err := internal.ExecHooks(cfg.Hooks.StartContainer); err != nil {
		return fmt.Errorf("execute startContainer hooks: %w", err)
	}

	// 8. TODO: Run the user-specified program from 'process' in the container
	// and update state to Running
	sockAddr := fmt.Sprintf("/tmp/brownie_%s.sock", containerID)
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("dial socket: %w", err)
	}

	fmt.Println("writing to socket...")
	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("send start over ipc: %w", err)
	}

	fmt.Println("reading from socket...")
	b, err := io.ReadAll(conn)
	if err != nil {
		return fmt.Errorf("reading response from socket: %w", err)
	}

	fmt.Println("res: ", string(b))

	// 9. Invoke poststart hooks
	if err := internal.ExecHooks(cfg.Hooks.PostStart); err != nil {
		return fmt.Errorf("execute poststart hooks: %w", err)
	}

	return nil
}
