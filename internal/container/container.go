package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nixpig/brownie/internal/bundle"
	"github.com/nixpig/brownie/internal/capabilities"
	"github.com/nixpig/brownie/internal/lifecycle"
	"github.com/nixpig/brownie/internal/namespace"
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
	SockAddr         string

	State *ContainerState
}

func New(id string, bundle *bundle.Bundle) (*Container, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id)
	if stat, err := os.Stat(path); stat != nil || os.IsExist(err) {
		return nil, fmt.Errorf("container with specified ID (%s) already exists", id)
	}

	if err := os.MkdirAll(path, os.ModeDir); err != nil {
		return nil, fmt.Errorf("create container directory: %w", err)
	}

	state, err := NewState(id, bundle)
	if err != nil {
		return nil, fmt.Errorf("create container state: %w", err)
	}

	specPath := filepath.Join(path, "config.json")
	if err := cp.Copy(bundle.SpecPath, specPath); err != nil {
		return nil, fmt.Errorf("copy spec from bundle to container: %w", err)
	}

	rootfsPath := filepath.Join(path, bundle.Spec.Root.Path)
	if err := cp.Copy(bundle.Rootfs, rootfsPath); err != nil {
		return nil, fmt.Errorf("copy rootfs from bundle to container: %w", err)
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
		ns := namespace.LinuxNamespace(ns)
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
	if spec.Process.Capabilities != nil {
		for _, cap := range spec.Process.Capabilities.Ambient {
			ambientCapsFlags = append(
				ambientCapsFlags,
				uintptr(capabilities.Capabilities[cap]),
			)
		}
	}

	return &Container{
		ID:       id,
		Path:     path,
		SpecPath: specPath,
		Spec:     &spec,
		Rootfs:   rootfsPath,
		SockAddr: filepath.Join(path, "container.sock"),

		NamespaceFlags:   namespaceFlags,
		UIDMappings:      uidMappings,
		GIDMappings:      gidMappings,
		AmbientCapsFlags: ambientCapsFlags,

		State: state,
	}, nil
}

func ForceClean(id string) error {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id)
	return os.RemoveAll(path)
}

func LoadContainer(id string) (*Container, error) {
	path := filepath.Join(pkg.BrownieRootDir, "containers", id)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("check container directory exists: %w", err)
	}

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
	if _, err := os.Stat(rootfsPath); err != nil {
		return nil, fmt.Errorf("check container rootfs exists: %w", err)
	}

	state, err := LoadState(id)
	if err != nil {
		return nil, fmt.Errorf("load state for container: %w", err)
	}

	return &Container{
		ID:       id,
		Path:     path,
		SpecPath: specPath,
		Spec:     &spec,
		Rootfs:   rootfsPath,
		SockAddr: filepath.Join(path, "container.sock"),

		State: state,
	}, nil
}

func (c *Container) ExecHooks(hook string) error {
	if c.Spec.Hooks == nil {
		return nil
	}

	var specHooks []specs.Hook
	switch hook {
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
	return c.State.Status == specs.StateCreated
}

func (c *Container) CanBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateStopped
}

func (c *Container) CanBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}
