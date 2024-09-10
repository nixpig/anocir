package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nixpig/brownie/internal"
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
				conn.Write([]byte(err.Error()))
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
	c, err := os.ReadFile(filepath.Join(containerPath, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(c, &cfg); err != nil {
		return fmt.Errorf("unmarshall config.json: %w", err)
	}

	containerRootfs := filepath.Join(containerPath, cfg.Root.Path)
	if err := internal.PivotRoot(containerRootfs); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}

		go server(conn, cfg)
	}
}
