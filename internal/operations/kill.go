// internal/operations/kill.go

package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
	"github.com/nixpig/anocir/internal/specconv"
)

type KillOpts struct {
	ID     string
	Signal string
}

func Kill(opts *KillOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Kill(specconv.SignalArgToSignal(opts.Signal)); err != nil {
		return fmt.Errorf("kill container: %w", err)
	}

	return nil
}
