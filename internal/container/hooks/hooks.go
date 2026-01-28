// Package hooks provides functionality for executing OCI container hooks.
package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// ExecHooks executes a list of OCI hooks, serialising the container state
// and passing it to each hook as standard input.
func ExecHooks(hooks []specs.Hook, state *specs.State) error {
	b, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	for _, h := range hooks {
		ctx := context.Background()

		binary, err := exec.LookPath(h.Path)
		if err != nil {
			return fmt.Errorf("find path of hook binary: %w", err)
		}

		if err := func() error {
			if h.Timeout != nil {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(
					ctx,
					time.Duration(*h.Timeout)*time.Second,
				)
				defer cancel()
			}

			cmd := exec.CommandContext(ctx, binary)

			cmd.Args = h.Args
			cmd.Env = h.Env
			cmd.Stdin = bytes.NewReader(b)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			slog.Debug(
				"execute hook",
				"container_id", state.ID,
				"path", h.Path,
				"args", h.Args,
				"env", h.Env,
				"timeout", h.Timeout,
			)

			if err := cmd.Run(); err != nil {
				return fmt.Errorf(
					"execute hook %s: %w\nstdout: %s\nstderr: %s",
					h.Path, err, stdout.String(), stderr.String(),
				)
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
}
