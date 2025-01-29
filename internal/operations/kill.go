// internal/operations/kill.go

package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
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

	if err := cntr.Kill(opts.Signal); err != nil {
		return fmt.Errorf("kill container: %w", err)
	}

	return nil
}
