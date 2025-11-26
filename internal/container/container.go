// Package container provides functionality for creating, running, and managing
// OCI-compliant containers.
package container

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nixpig/anocir/internal/container/hooks"
	"github.com/nixpig/anocir/internal/terminal"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// containerSockFilename is the filename of the socket used by the runtime to
// send messages to the container.
const containerSockFilename = "c.sock"

// Container represents an OCI container, including its state, specification,
// and other runtime details.
type Container struct {
	State           *specs.State
	ConsoleSocket   string
	ConsoleSocketFD int

	spec          *specs.Spec
	pty           *terminal.Pty
	pidFile       string
	rootDir       string
	containerSock string
	logFile       string
	lockFile      *os.File
}

// ContainerOpts holds the options for creating a new Container.
type ContainerOpts struct {
	ID            string
	Bundle        string
	Spec          *specs.Spec
	ConsoleSocket string
	PIDFile       string
	RootDir       string
	LogFile       string
}

// New creates a Container based on the provided opts and saves its state.
// The Container will be in the 'creating' state.
func New(opts *ContainerOpts) *Container {
	state := &specs.State{
		Version:     specs.Version,
		ID:          opts.ID,
		Bundle:      opts.Bundle,
		Annotations: opts.Spec.Annotations,
		Status:      specs.StateCreating,
	}

	return &Container{
		State:         state,
		spec:          opts.Spec,
		ConsoleSocket: opts.ConsoleSocket,
		pidFile:       opts.PIDFile,

		rootDir: opts.RootDir,
		logFile: opts.LogFile,
		containerSock: filepath.Join(
			opts.RootDir,
			opts.ID,
			containerSockFilename,
		),
	}
}

// execHooks executes the hooks for the given phase of the Container execution.
func (c *Container) execHooks(phase Lifecycle) error {
	if c.spec.Hooks == nil {
		return nil
	}

	var h []specs.Hook

	switch phase {
	case LifecycleCreateRuntime:
		h = append(h, c.spec.Hooks.CreateRuntime...)
	case LifecycleCreateContainer:
		h = append(h, c.spec.Hooks.CreateContainer...)
	case LifecycleStartContainer:
		h = append(h, c.spec.Hooks.StartContainer...)
	case LifecyclePrestart:
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker.
		h = append(h, c.spec.Hooks.Prestart...)
	case LifecyclePoststart:
		h = append(h, c.spec.Hooks.Poststart...)
	case LifecyclePoststop:
		h = append(h, c.spec.Hooks.Poststop...)
	}

	if len(h) > 0 {
		if err := hooks.ExecHooks(h, c.State); err != nil {
			return err
		}
	}

	return nil
}

// rootFS returns the path to the Container root filesystem.
func (c *Container) rootFS() string {
	if strings.HasPrefix(c.spec.Root.Path, "/") {
		return c.spec.Root.Path
	}

	return filepath.Join(c.State.Bundle, c.spec.Root.Path)
}

func (c *Container) canBeDeleted() bool {
	return c.State.Status == specs.StateStopped
}

func (c *Container) canBeStarted() bool {
	return c.State.Status == specs.StateCreated
}

func (c *Container) canBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateCreated
}
