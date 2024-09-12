package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func server(conn net.Conn, spec specs.Spec) error {
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
			cmd := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
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

func Fork(containerID, initSockAddr, containerSockAddr string) error {
	if err := os.RemoveAll(containerSockAddr); err != nil {
		return fmt.Errorf("remove existing socket: %w", err)
	}

	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", containerID)
	configJson, err := os.ReadFile(filepath.Join(containerPath, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJson, &spec); err != nil {
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	listener, err := net.Listen("unix", containerSockAddr)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	defer listener.Close()

	initConn, err := net.Dial("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("dialing init socket: %w", err)
	}
	defer initConn.Close()

	containerRootfs := filepath.Join(containerPath, spec.Root.Path)

	if err := filesystem.MountProc(containerRootfs); err != nil {
		return fmt.Errorf("mount proc: %w", err)
	}

	if err := filesystem.MountDefaultDevices(containerRootfs); err != nil {
		return fmt.Errorf("mount dev: %w", err)
	}

	if err := filesystem.MountRootfs(containerRootfs); err != nil {
		return fmt.Errorf("mount rootfs: %w", err)
	}

	if err := filesystem.PivotRootfs(containerRootfs); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	if n, err := initConn.Write([]byte("ready")); n == 0 || err != nil {
		return errors.New("failed to write to init socket")
	}

	for {
		containerConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept connection: %w", err)
		}

		go server(containerConn, spec)
	}
}
