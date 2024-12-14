package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/nixpig/brownie/lifecycle"
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/mod/semver"
)

const (
	initSockFilename      = "init.sock"
	containerSockFilename = "container.sock"
	OCIVersion            = "1.0.1-dev"
	containerRootDir      = "/var/lib/brownie/containers"
	stateFilename         = "state.json"
	configFilename        = "config.json"
)

type Container struct {
	State *ContainerState
	Spec  *specs.Spec
	Opts  *ContainerOpts

	termFD *int
}

type ContainerState struct {
	Version       string               `json:"ociVersion"`
	ID            string               `json:"id"`
	Bundle        string               `json:"bundle"`
	Annotations   map[string]string    `json:"annotations"`
	Status        specs.ContainerState `json:"status"`
	PID           int                  `json:"pid"`
	ConsoleSocket *int                 `json:"consoleSocket"`
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
	b, err := os.ReadFile(filepath.Join(bundle, configFilename))
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
		return nil, err
	}

	if !semver.IsValid("v" + spec.Version) {
		// TODO: rollback state?
		return nil, fmt.Errorf("invalid version: %s", spec.Version)
	}

	state := &ContainerState{
		Version:     OCIVersion,
		ID:          id,
		Bundle:      absBundlePath,
		Annotations: spec.Annotations,
		Status:      specs.StateCreating,
	}

	cntr := Container{
		State: state,
		Spec:  spec,
		Opts:  opts,
	}

	if err := os.MkdirAll(
		filepath.Join(containerRootDir, cntr.ID()),
		0644,
	); err != nil {
		return nil, fmt.Errorf("create container directory: %w", err)
	}

	if err := cntr.Save(); err != nil {
		return nil, fmt.Errorf("save newly created container: %w", err)
	}

	return &cntr, nil
}

func Load(id string) (*Container, error) {
	s, err := os.ReadFile(filepath.Join(containerRootDir, id, stateFilename))
	if err != nil {
		return nil, err
	}

	state := ContainerState{}
	if err := json.Unmarshal(s, &state); err != nil {
		return nil, err
	}

	bundle := state.Bundle
	c, err := os.ReadFile(filepath.Join(bundle, configFilename))
	if err != nil {
		return nil, err
	}

	conf := specs.Spec{}
	if err := json.Unmarshal(c, &conf); err != nil {
		return nil, fmt.Errorf("unmarshall state to struct: %w", err)
	}

	cntr := &Container{
		State: &state,
		Spec:  &conf,
	}

	if err := cntr.RefreshState(); err != nil {
		return nil, err
	}

	return cntr, nil
}

func (c *Container) RefreshState() error {
	b, err := os.ReadFile(filepath.Join(containerRootDir, c.ID(), stateFilename))
	if err != nil {
		return fmt.Errorf("refresh from state file: %w", err)
	}

	if err := json.Unmarshal(b, c.State); err != nil {
		return fmt.Errorf("unmarshal refreshed state: %w", err)
	}

	process, err := os.FindProcess(c.State.PID)
	if err != nil {
		return err
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		c.SetStatus(specs.StateStopped)
		if err := c.Save(); err != nil {
			return fmt.Errorf("(refresh) save container state: %w", err)
		}
	}

	return nil
}

func (c *Container) Save() error {
	b, err := json.Marshal(c.State)
	if err != nil {
		return err
	}

	if err := os.WriteFile(
		filepath.Join(containerRootDir, c.ID(), stateFilename),
		b,
		0644,
	); err != nil {
		return fmt.Errorf("write state file (%s): %w", c.State.Status, err)
	}

	if c.Opts != nil && c.Opts.PIDFile != "" {
		if err := os.WriteFile(
			c.Opts.PIDFile,
			[]byte(strconv.Itoa(c.PID())),
			0666,
		); err != nil {
			return fmt.Errorf("write pid to file (%s): %w", c.Opts.PIDFile, err)
		}
	}

	return nil
}

func (c *Container) ExecHooks(lifecycleHook string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var specHooks []specs.Hook
	switch lifecycleHook {
	case "prestart":
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker
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

	s, err := json.Marshal(c.State)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	return lifecycle.ExecHooks(specHooks, string(s))
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
	if strings.HasPrefix(c.Spec.Root.Path, "/") {
		return c.Spec.Root.Path
	}

	return filepath.Join(c.Bundle(), c.Spec.Root.Path)
}
