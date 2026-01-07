package container

import (
	"github.com/nixpig/anocir/internal/container/hooks"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// Lifecycle represents the container lifecycle stages that hooks are
// executed on.
type Lifecycle string

const (
	LifecycleCreateRuntime   Lifecycle = "createRuntime"
	LifecycleCreateContainer Lifecycle = "createContainer"
	LifecycleStartContainer  Lifecycle = "startContainer"
	LifecyclePrestart        Lifecycle = "prestart"
	LifecyclePoststart       Lifecycle = "poststart"
	LifecyclePoststop        Lifecycle = "poststop"
)

// execHooks executes the hooks for the given phase of the Container execution.
func (c *Container) execHooks(phase Lifecycle) error {
	if c.spec.Hooks == nil {
		return nil
	}

	var h []specs.Hook

	switch phase {
	case LifecycleCreateRuntime:
		h = c.spec.Hooks.CreateRuntime
	case LifecycleCreateContainer:
		h = c.spec.Hooks.CreateContainer
	case LifecycleStartContainer:
		h = c.spec.Hooks.StartContainer
	case LifecyclePrestart:
		//lint:ignore SA1019 marked as deprecated, but still required by OCI Runtime integration tests and used by other tools like Docker.
		h = c.spec.Hooks.Prestart
	case LifecyclePoststart:
		h = c.spec.Hooks.Poststart
	case LifecyclePoststop:
		h = c.spec.Hooks.Poststop
	}

	if len(h) > 0 {
		if err := hooks.ExecHooks(h, c.State); err != nil {
			return err
		}
	}

	return nil
}
