// internal/operations/kill.go

package operations

import (
	"fmt"
	"strconv"

	"github.com/nixpig/anocir/internal/container"
	"golang.org/x/sys/unix"
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

	sig, err := strconv.Atoi(opts.Signal)
	if err != nil {
		return fmt.Errorf("convert signal to int: %w", err)
	}

	if err := cntr.Kill(unix.Signal(sig)); err != nil {
		return fmt.Errorf("kill container: %w", err)
	}

	if err := cntr.Save(); err != nil {
		return fmt.Errorf("save container: %w", err)
	}

	return nil
}
