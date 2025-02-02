package operations

import (
	"fmt"

	"github.com/nixpig/anocir/internal/container"
)

type DeleteOpts struct {
	ID    string
	Force bool
}

func Delete(opts *DeleteOpts) error {
	cntr, err := container.Load(opts.ID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if err := cntr.Delete(opts.Force); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	return nil
}
