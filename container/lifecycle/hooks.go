package lifecycle

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func ExecHooks(hooks []specs.Hook) error {
	for _, h := range hooks {

		ctx := context.Background()
		if h.Timeout != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(
				ctx,
				time.Duration(*h.Timeout)*time.Second,
			)
			defer cancel()
		}
		cmd := exec.CommandContext(ctx, h.Path, h.Args[1:]...)
		cmd.Env = h.Env

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("exec hook: %s %+v: %w", h.Path, h.Args, err)
		}
	}

	return nil
}
