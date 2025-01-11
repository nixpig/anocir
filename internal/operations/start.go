// internal/operations/start.go

package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

type StartOpts struct {
	ID string
}

func Start(opts *StartOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Start(); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	if err := cntr.Save(); err != nil {
		return fmt.Errorf("save container: %w", err)
	}

	return nil
}
