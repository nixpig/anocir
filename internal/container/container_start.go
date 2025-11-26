package container

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container/ipc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// Start begins the execution of the Container. It executes pre-start and
// post-start hooks and sends the "start" message to the runtime process.
func (c *Container) Start() error {
	if c.spec.Process == nil {
		c.State.Status = specs.StateStopped
		if err := c.Save(); err != nil {
			return fmt.Errorf("save state stopped: %w", err)
		}
		// Nothing to do; silent return.
		return nil
	}

	if !c.canBeStarted() {
		return fmt.Errorf(
			"container cannot be started in current state (%s)",
			c.State.Status,
		)
	}

	if err := c.execHooks(LifecyclePrestart); err != nil {
		return fmt.Errorf("execute prestart hooks: %w", err)
	}

	containerSock := ipc.NewSocket(c.containerSock)

	conn, err := containerSock.Dial()
	if err != nil {
		return fmt.Errorf("dial container sock: %w", err)
	}

	if err := ipc.SendMessage(conn, ipc.StartMsg); err != nil {
		return fmt.Errorf(
			"write '%s' msg to container sock: %w",
			ipc.StartMsg,
			err,
		)
	}
	defer conn.Close()

	c.State.Status = specs.StateRunning
	if err := c.Save(); err != nil {
		return fmt.Errorf("save state running: %w", err)
	}

	if err := c.execHooks(LifecyclePoststart); err != nil {
		return fmt.Errorf("exec poststart hooks: %w", err)
	}

	return nil
}

func (c *Container) canBeStarted() bool {
	return c.State.Status == specs.StateCreated
}
