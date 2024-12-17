package lifecycle

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
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

		binary, err := exec.LookPath(h.Path)
		if err != nil {
			return fmt.Errorf("find hook binary: %w", err)
		}

		path := filepath.Dir(h.Path)

		cmd := exec.CommandContext(ctx, binary, path)

		cmd.Args = append(h.Args, state)
		cmd.Env = h.Env
		cmd.Stdin = strings.NewReader(state)

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
