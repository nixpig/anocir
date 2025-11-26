// Package container provides functionality for creating, running, and managing
// OCI-compliant containers.
package container

import (
	"os"
	"path/filepath"
	"strings"

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
	RootDir         string

	spec          *specs.Spec
	pty           *terminal.Pty
	pidFile       string
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

		RootDir: opts.RootDir,
		logFile: opts.LogFile,
		containerSock: filepath.Join(
			opts.RootDir,
			opts.ID,
			containerSockFilename,
		),
	}
}

// rootFS returns the path to the Container root filesystem.
func (c *Container) rootFS() string {
	if strings.HasPrefix(c.spec.Root.Path, "/") {
		return c.spec.Root.Path
	}

	return filepath.Join(c.State.Bundle, c.spec.Root.Path)
}

func (c *Container) stateFilepath() string {
	return filepath.Join(c.RootDir, c.State.ID, "state.json")
}
