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

		args := append(h.Args, state)
		cmd := exec.CommandContext(ctx, h.Path, args...)

		// I don't know why these need to be set here
		// I thought they're set in the CommandContext above, but maybe not.
		// -- Skill issue
		cmd.Path = h.Path
		cmd.Args = args
		// ---

		cmd.Env = h.Env
		cmd.Stdin = strings.NewReader(state)

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
