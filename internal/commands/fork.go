package commands

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nixpig/brownie/internal/filesystem"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func server(conn net.Conn, spec specs.Spec) {
	defer conn.Close()

	b := make([]byte, 128)

	for {
		n, err := conn.Read(b)
		if err != nil {
			// TODO: log it
			fmt.Println(fmt.Errorf("read from connection: %s", err))
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
			return
		}
	}
}

func Fork(containerID, initSockAddr, containerSockAddr string) {
	initConn, err := net.Dial("unix", initSockAddr)
	if err != nil {
		// TODO: log it
		fmt.Println(fmt.Errorf("dialing init socket: %s", err))
		return
	}
	defer initConn.Close()

	if err := os.RemoveAll(containerSockAddr); err != nil {
		initConn.Write([]byte(fmt.Sprintf("remove socket: %s", err)))
		return
	}

	containerPath := filepath.Join(pkg.BrownieRootDir, "containers", containerID)
	configJSON, err := os.ReadFile(filepath.Join(containerPath, "config.json"))
	if err != nil {
		initConn.Write([]byte(fmt.Sprintf("read config file: %s", err)))
		return
	}

	var spec specs.Spec
	if err := json.Unmarshal(configJSON, &spec); err != nil {
		initConn.Write([]byte(fmt.Sprintf("unmarshal config.json: %s", err)))
		return
	}

	listener, err := net.Listen("unix", containerSockAddr)
	if err != nil {
		initConn.Write([]byte(fmt.Sprintf("listen on socket: %s", err)))
		return
	}
	defer listener.Close()

	containerRootfs := filepath.Join(containerPath, spec.Root.Path)

	if err := filesystem.MountProc(containerRootfs); err != nil {
		initConn.Write([]byte(fmt.Sprintf("mount proc: %s", err)))
	}

	if spec.Linux != nil && len(spec.Linux.Devices) > 0 {
		for _, dev := range spec.Linux.Devices {
			target := filepath.Join(containerRootfs, strings.TrimPrefix(dev.Path, "/"))
			fmt.Printf("mount '%s' to '%s': \n", dev.Path, target)
			// if err := syscall.Mount(
			// 	target,
			// 	target,
			// 	unix.MS_BIND,
			// 	dev.FileMode, // ???
			// ); err != nil {
			// 	// TODO: log error
			// 	initConn.Write([]byte(err.Error()))
			// }
		}

	}

	if err := filesystem.MountDefaultDevices(containerRootfs); err != nil {
		initConn.Write([]byte(fmt.Sprintf("mount dev: %s", err)))
		return
	}

	if err := filesystem.MountRootfs(containerRootfs); err != nil {
		initConn.Write([]byte(fmt.Sprintf("mount rootfs: %s", err)))
		return
	}

	if err := filesystem.PivotRootfs(containerRootfs); err != nil {
		initConn.Write([]byte(fmt.Sprintf("pivot root: %s", err)))
		return
	}

	if n, err := initConn.Write([]byte("ready")); n == 0 || err != nil {
		// TODO: log error
		return
	}

	for {
		containerConn, err := listener.Accept()
		if err != nil {
			initConn.Write([]byte(fmt.Sprintf("accept connection: %s", err)))
			continue
		}

		go server(containerConn, spec)
	}
}
