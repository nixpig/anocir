package container

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/rs/zerolog"
)

func (c *Container) Kill(sig syscall.Signal, log *zerolog.Logger) error {
	if !c.CanBeKilled() {
		log.Error().Str("state", string(c.State.Status)).Msg("container cannot be killed in current state")
		return fmt.Errorf("container cannot be killed in current state: %s", c.Status())
	}

	if err := syscall.Kill(c.PID(), sig); err != nil {
		log.Error().Err(err).Int("pid", c.PID()).Msg("failed to execute kill syscall")
		return fmt.Errorf("failed to execute kill syscall (process: %d): %w", c.PID(), err)
	}

	c.SetStatus(specs.StateStopped)
	if err := c.HSave(); err != nil {
		log.Error().Err(err).Msg("failed to save stopped state")
		return fmt.Errorf("failed to save stopped state: %w", err)
	}

	// TODO: delete everything then
	if err := c.ExecHooks("poststop"); err != nil {
		log.Error().Err(err).Msg("failed to execute poststop hooks")
		fmt.Println("failed to execute poststop hooks")
		// TODO: log a warning???
	}

	log.Info().Msg("💛 sent the killsig and exiting with nil")

	return nil
}
