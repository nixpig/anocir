package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func ExecHooks(hooks []specs.Hook, state *specs.State) error {
	s, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

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
			return fmt.Errorf("find path of hook binary: %w", err)
		}

		path := filepath.Dir(h.Path)

		cmd := exec.CommandContext(ctx, binary, path)

		cmd.Args = append(h.Args, string(s))
		cmd.Env = h.Env
		cmd.Stdin = strings.NewReader(string(s))

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("execute hook %s: %w", h.Path, err)
		}
	}

	return nil
}
