package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/pkg"
	"github.com/opencontainers/runtime-spec/specs-go"
	cp "github.com/otiai10/copy"
)

type Container struct {
	ID               string
	Path             string
	Rootfs           string
	SpecPath         string
	Spec             *specs.Spec
	NamespaceFlags   uintptr
	UIDMappings      []syscall.SysProcIDMap
	GIDMappings      []syscall.SysProcIDMap
	AmbientCapsFlags []uintptr

	State *ContainerState
}

type ContainerState struct {
	Path string
	specs.State
}

func NewContainerState(id string, bundle *Bundle) (*ContainerState, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id, "state.json")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create state file: %w", err)
	}
	if f != nil {
		f.Close()
	}

	return &ContainerState{
		Path: path,
		State: specs.State{
			Version:     bundle.Spec.Version,
			ID:          id,
			Bundle:      bundle.Path,
			Annotations: bundle.Spec.Annotations,
		},
	}, nil
}

func LoadContainerState(id string) (*ContainerState, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id, "state.json")
	b, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("read container state file: %w", err)
	}

	var state ContainerState

	if err := json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state file: %w", err)
	}

	return &state, nil
}

func (c *ContainerState) Set(status specs.ContainerState) {
	c.Status = status
}

func (c *ContainerState) Save() error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err := os.WriteFile(c.Path, b, 0644); err != nil {
		return err
	}

	return nil
}

func NewContainer(id string, bundle *Bundle) (*Container, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id)
	if stat, _ := os.Stat(path); stat != nil {
		return nil, errors.New("container with specified ID already exists")
	}

	if err := os.MkdirAll(path, os.ModeDir); err != nil {
		return nil, fmt.Errorf("make container directory: %w", err)
	}

	state, err := NewContainerState(id, bundle)
	if err != nil {
		return nil, fmt.Errorf("create container state: %w", err)
	}

	specPath := filepath.Join(path, "config.json")
	if err := cp.Copy(bundle.SpecPath, specPath); err != nil {
		return nil, fmt.Errorf("copy bundle spec to container spec: %w", err)
	}

	rootfsPath := filepath.Join(path, bundle.Spec.Root.Path)
	if err := cp.Copy(bundle.Rootfs, rootfsPath); err != nil {
		return nil, fmt.Errorf("copy bundle rootfs to container rootfs: %w", err)
	}

	specJSON, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec from container: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	var namespaceFlags uintptr
	for _, ns := range spec.Linux.Namespaces {
		ns := LinuxNamespace(ns)
		flag, err := ns.ToFlag()
		if err != nil {
			return nil, fmt.Errorf("convert namespace to flag: %w", err)
		}

		namespaceFlags |= flag
	}

	var uidMappings []syscall.SysProcIDMap
	var gidMappings []syscall.SysProcIDMap
	if spec.Process != nil {
		namespaceFlags |= syscall.CLONE_NEWUSER

		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: int(spec.Process.User.UID),
			HostID:      os.Geteuid(),
			Size:        1,
		})

		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: int(spec.Process.User.GID),
			HostID:      os.Getegid(),
			Size:        1,
		})
	}

	for _, uidMapping := range spec.Linux.UIDMappings {
		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: int(uidMapping.ContainerID),
			HostID:      int(uidMapping.HostID),
			Size:        int(uidMapping.Size),
		})
	}

	for _, gidMapping := range spec.Linux.GIDMappings {
		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: int(gidMapping.ContainerID),
			HostID:      int(gidMapping.HostID),
			Size:        int(gidMapping.Size),
		})
	}

	var ambientCapsFlags []uintptr
	for _, cap := range spec.Process.Capabilities.Ambient {
		ambientCapsFlags = append(ambientCapsFlags, uintptr(capabilities.Capabilities[cap]))
	}

	return &Container{
		ID:               id,
		Path:             path,
		SpecPath:         specPath,
		Spec:             &spec,
		Rootfs:           rootfsPath,
		NamespaceFlags:   namespaceFlags,
		UIDMappings:      uidMappings,
		GIDMappings:      gidMappings,
		AmbientCapsFlags: ambientCapsFlags,

		State: state,
	}, nil
}

func LoadContainer(id string) (*Container, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id)
	specPath := filepath.Join(path, "config.json")
	specJSON, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec from container: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	rootfsPath := filepath.Join(path, spec.Root.Path)

	state, err := LoadContainerState(id)
	if err != nil {
		return nil, fmt.Errorf("load state for container: %w", err)
	}

	return &Container{
		ID:       id,
		Path:     path,
		SpecPath: specPath,
		Spec:     &spec,
		Rootfs:   rootfsPath,

		State: state,
	}, nil
}

func (c *Container) ExecHooks(lifecycle string) error {
	var hooks []specs.Hook

	switch lifecycle {
	case "createRuntime":
		hooks = c.Spec.Hooks.CreateRuntime
	case "createContainer":
		hooks = c.Spec.Hooks.CreateContainer
	}

	return ExecHooks(hooks)
}
