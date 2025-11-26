package container

import (
	"fmt"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// Kill sends the given sig to the Container process.
func (c *Container) Kill(sig string) error {
	if !c.canBeKilled() {
		return fmt.Errorf(
			"container cannot be killed in current state (%s)",
			c.State.Status,
		)
	}

	if err := platform.SendSignal(
		c.State.Pid,
		platform.ParseSignal(sig),
	); err != nil {
		return fmt.Errorf(
			"send signal '%s' to process '%d': %w",
			sig,
			c.State.Pid,
			err,
		)
	}

	return nil
}

func (c *Container) canBeKilled() bool {
	return c.State.Status == specs.StateRunning ||
		c.State.Status == specs.StateCreated
}
