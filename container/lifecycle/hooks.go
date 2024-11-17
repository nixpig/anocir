package lifecycle

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func ExecHooks(hooks []specs.Hook, state string) error {
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

		args := h.Args[1:]
		args = append(args, state)
		cmd := exec.CommandContext(ctx, h.Path, args...)
		cmd.Env = h.Env

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start exec hook: %s %+v: %w", h.Path, h.Args, err)
		}

		if _, err := stdin.Write([]byte(state)); err != nil {
			return fmt.Errorf("write state to stdin: %w", err)
		}
		stdin.Close()

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("wait exec hook: %s %+v: %w", h.Path, h.Args, err)
		}
	}

	return nil
}
