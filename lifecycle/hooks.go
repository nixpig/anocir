package lifecycle

import (
	"context"
	"os/exec"
	"strings"
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
		cmd.Stdin = strings.NewReader(state)

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
