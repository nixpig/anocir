package container

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

const (
	containerRootDir      = "/var/lib/anocir/containers"
	initSockFilename      = "init.sock"
	containerSockFilename = "container.sock"
)

type Container struct {
	State *specs.State
	Spec  *specs.Spec
}

type NewContainerOpts struct {
	ID     string
	Bundle string
	Spec   *specs.Spec
}

func New(opts *NewContainerOpts) (*Container, error) {
	if exists(opts.ID) {
		return nil, fmt.Errorf("container '%s' exists", opts.ID)
	}

	state := &specs.State{
		Version:     specs.Version,
		ID:          opts.ID,
		Bundle:      opts.Bundle,
		Annotations: opts.Spec.Annotations,
		Status:      specs.StateCreating,
	}

	c := &Container{
		State: state,
		Spec:  opts.Spec,
	}

	return c, nil
}

func (c *Container) Save() error {
	if err := os.MkdirAll(
		filepath.Join(containerRootDir, c.State.ID),
		0666,
	); err != nil {
		return fmt.Errorf("create container directory: %w", err)
	}

	state, err := json.Marshal(c.State)
	if err != nil {
		return fmt.Errorf("serialise container state: %w", err)
	}

	if err := os.WriteFile(
		filepath.Join(containerRootDir, c.State.ID, "state.json"),
		state,
		0666,
	); err != nil {
		return fmt.Errorf("write container state: %w", err)
	}

	return nil
}

func (c *Container) Init() error {
	cmd := exec.Command("/proc/self/exe", "reexec", c.State.ID)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	initSockAddr := filepath.Join(
		containerRootDir,
		c.State.ID,
		initSockFilename,
	)

	listener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("listen on init sock: %w", err)
	}
	defer listener.Close()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reexec container process: %w", err)
	}

	c.State.Pid = cmd.Process.Pid

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release container process: %w", err)
	}

	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept on init sock: %w", err)
	}
	defer conn.Close()

	b := make([]byte, 128)
	n, err := conn.Read(b)
	if err != nil {
		return fmt.Errorf("read bytes from init sock connection: %w", err)
	}

	msg := string(b[:n])
	if msg != "ready" {
		return fmt.Errorf("expecting 'ready' but received '%s'", msg)
	}

	c.State.Status = specs.StateCreated

	return nil
}

func (c *Container) Reexec() error {
	initConn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, initSockFilename),
	)
	if err != nil {
		return fmt.Errorf("dial init sock: %w", err)
	}

	if _, err := initConn.Write([]byte("ready")); err != nil {
		return fmt.Errorf("write 'ready' msg to init sock: %w", err)
	}
	// close immediately, rather than defering
	initConn.Close()

	listener, err := net.Listen(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, containerSockFilename),
	)
	if err != nil {
		return fmt.Errorf("listen on container sock: %w", err)
	}

	containerConn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("accept on container sock: %w", err)
	}

	b := make([]byte, 128)
	n, err := containerConn.Read(b)
	if err != nil {
		return fmt.Errorf("read bytes from container sock: %w", err)
	}

	msg := string(b[:n])
	if msg != "start" {
		return fmt.Errorf("expecting 'start' but received '%s'", msg)
	}

	containerConn.Close()
	listener.Close()

	bin, err := exec.LookPath(c.Spec.Process.Args[0])
	if err != nil {
		return fmt.Errorf("find path of user process binary: %w", err)
	}

	args := c.Spec.Process.Args
	env := os.Environ()

	if err := syscall.Exec(bin, args, env); err != nil {
		return fmt.Errorf("execve (%s, %s, %v): %w", bin, args, env, err)
	}

	panic("if you got here then something went horribly wrong")
}

func (c *Container) Start() error {
	if c.Spec.Process == nil {
		// nothing to do; silent return
		return nil
	}

	if !c.canBeStarted() {
		return fmt.Errorf("container cannot be started in current state (%s)", c.State.Status)
	}

	conn, err := net.Dial(
		"unix",
		filepath.Join(containerRootDir, c.State.ID, containerSockFilename),
	)
	if err != nil {
		return fmt.Errorf("dial container sock: %w", err)
	}

	if _, err := conn.Write([]byte("start")); err != nil {
		return fmt.Errorf("write 'start' msg to container sock: %w", err)
	}
	conn.Close()

	c.State.Status = specs.StateRunning

	return nil
}

func (c *Container) Delete(force bool) error {
	if !force && !c.canBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state (%s) try using '--force'", c.State.Status)
	}

	process, err := os.FindProcess(c.State.Pid)
	if err != nil {
		return fmt.Errorf("find container process to delete: %w", err)
	}
	if process != nil {
		process.Signal(unix.SIGKILL)
	}

	if err := os.RemoveAll(
		filepath.Join(containerRootDir, c.State.ID),
	); err != nil {
		return fmt.Errorf("delete container directory: %w", err)
	}

	return nil
}

func (c *Container) canBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}

func (c *Container) canBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func Load(id string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(containerRootDir, id, "state.json"))
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state *specs.State
	if err := json.Unmarshal(s, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	config, err := os.ReadFile(filepath.Join(state.Bundle, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(config, &spec); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	c := &Container{
		State: state,
		Spec:  spec,
	}

	return c, nil
}

func exists(containerID string) bool {
	_, err := os.Stat(filepath.Join(containerRootDir, containerID))

	return err == nil
}
