package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	containerRootDir = "/var/lib/anocir/containers"
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

func (c *Container) Delete(force bool) error {
	if !force && !c.canBeDeleted() {
		return fmt.Errorf("container cannot be deleted in current state (%s) try using '--force'", c.State.Status)
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
