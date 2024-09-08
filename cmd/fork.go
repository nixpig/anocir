package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nixpig/brownie/pkg"
	"github.com/nixpig/brownie/pkg/config"
)

func server(conn net.Conn, cfg config.Config) error {
	defer conn.Close()

	b := make([]byte, 128)

	for {
		n, err := conn.Read(b)
		if err != nil {
			return fmt.Errorf("read from connection: %w", err)
		}

		if n == 0 {
			break
		}

		switch string(b[:n]) {
		case "start":
			conn.Write([]byte("execute command in here!!"))
			conn.Write([]byte(fmt.Sprintf("%s %s", cfg.Process.Args[0], cfg.Process.Args[1:])))
			cmd := exec.Command(cfg.Process.Args[0], cfg.Process.Args[1:]...)
			cmd.Stdout = conn
			cmd.Stderr = conn
			if err := cmd.Run(); err != nil {
				conn.Write([]byte("done fucked up!"))
			}
			return nil
		}
	}

	return nil
}

func Fork(containerID, bundlePath string) error {
	sockAddr := fmt.Sprintf("/tmp/brownie_%s.sock", containerID)

	if err := os.RemoveAll(sockAddr); err != nil {
		return fmt.Errorf("remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	defer listener.Close()

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

	c, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(c, &cfg); err != nil {
		return fmt.Errorf("unmarshall config.json: %w", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}

		go server(conn, cfg)
	}
}
