package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nixpig/brownie/container/lifecycle"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
)

const initSockFilename = "init.sock"
const containerSockFilename = "container.sock"

type Container struct {
	State *ContainerState
	Spec  *specs.Spec
	Opts  *ContainerOpts

	termFD  *int
	initIPC ipcCtrl
}

type ContainerState struct {
	Version     string
	ID          string
	Bundle      string
	Annotations map[string]string
	Status      specs.ContainerState
	PID         int
}

type ipcCtrl struct {
	ch     chan []byte
	closer func() error
}

type ContainerOpts struct {
	PIDFile       string
	ConsoleSocket string
	Stdin         *os.File
	Stdout        *os.File
	Stderr        *os.File
}

func New(
	id string,
	bundle string,
	opts *ContainerOpts,
) (*Container, error) {
	b, err := os.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("read container config: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, fmt.Errorf("parse container config: %w", err)
	}

	if spec.Linux == nil {
		return nil, errors.New("only linux containers are supported")
	}

	if spec.Root == nil {
		return nil, errors.New("root is required")
	}

	absBundlePath, err := filepath.Abs(bundle)
	if err != nil {
		return nil, fmt.Errorf("construct absolute bundle path: %w", err)
	}

	state := &ContainerState{
		Version:     pkg.OCIVersion,
		ID:          id,
		Bundle:      absBundlePath,
		Annotations: map[string]string{},
		Status:      specs.StateCreating,
	}

	cntr := Container{
		State: state,
		Spec:  spec,
		Opts:  opts,
	}

	if err := cntr.HSave(); err != nil {
		return nil, fmt.Errorf("save newly created container: %w", err)
	}

	return &cntr, nil
}

func Load(bundle string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(bundle, "state.json"))
	if err != nil {
		return nil, err
	}

	state := ContainerState{}
	if err := json.Unmarshal([]byte(s), &state); err != nil {
		return nil, err
	}

	c, err := os.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return nil, err
	}

	conf := specs.Spec{}
	if err := json.Unmarshal([]byte(c), &conf); err != nil {
		return nil, fmt.Errorf("unmarshall state to struct: %w", err)
	}

	cntr := &Container{
		State: &state,
		Spec:  &conf,
	}

	if err := cntr.RefreshState(); err != nil {
		return nil, fmt.Errorf("refresh state: %w", err)
	}

	return cntr, nil
}

func (c *Container) RefreshState() error {
	b, err := os.ReadFile(filepath.Join(c.Bundle(), "state.json"))
	if err != nil {
		return fmt.Errorf("refresh from state file: %w", err)
	}

	if err := json.Unmarshal(b, c.State); err != nil {
		return fmt.Errorf("unmarshall refreshed state: %w", err)
	}

	return nil
}

func (c *Container) Save(configPath string) error {
	b, err := json.Marshal(c.State)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, b, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func (c *Container) CSave() error {
	return c.Save("/state.json")
}

func (c *Container) HSave() error {
	return c.Save(filepath.Join(c.Bundle(), "state.json"))
}

func (c *Container) ExecHooks(lifecycleHook string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var specHooks []specs.Hook
	switch lifecycleHook {
	case "prestart":
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests
		specHooks = c.Spec.Hooks.Prestart
	case "createRuntime":
		specHooks = c.Spec.Hooks.CreateRuntime
	case "createContainer":
		specHooks = c.Spec.Hooks.CreateContainer
	case "startContainer":
		specHooks = c.Spec.Hooks.StartContainer
	case "poststart":
		specHooks = c.Spec.Hooks.Poststart
	case "poststop":
		specHooks = c.Spec.Hooks.Poststop
	}

	return lifecycle.ExecHooks(specHooks)
}

func (c *Container) CanBeStarted() bool {
	return c.Status() == specs.StateCreated
}

func (c *Container) CanBeKilled() bool {
	return c.Status() == specs.StateRunning ||
		c.Status() == specs.StateCreated
}

func (c *Container) CanBeDeleted() bool {
	return c.Status() == specs.StateStopped
}

func (c *Container) SetStatus(status specs.ContainerState) {
	c.State.Status = status
}

func (c *Container) Status() specs.ContainerState {
	return c.State.Status
}

func (c *Container) SetPID(pid int) {
	c.State.PID = pid
}

func (c *Container) PID() int {
	return c.State.PID
}

func (c *Container) SetBundle(bundle string) {
	c.State.Bundle = bundle
}

func (c *Container) Bundle() string {
	return c.State.Bundle
}

func (c *Container) SetID(id string) {
	c.State.ID = id
}

func (c *Container) ID() string {
	return c.State.ID
}

func (c *Container) Rootfs() string {
	if strings.Index(c.Spec.Root.Path, "/") == 0 {
		return c.Spec.Root.Path
	}

	return filepath.Join(c.Bundle(), c.Spec.Root.Path)
}
