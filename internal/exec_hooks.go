package internal

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/nixpig/brownie/pkg/config"
)

func ExecHooks(hooks []config.Hook) error {
	for _, h := range hooks {
		ctx := context.Background()
		var cancel context.CancelFunc
		if h.Timeout != nil {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(*h.Timeout)*time.Second)
			defer cancel()
		}
		cmd := exec.CommandContext(ctx, h.Path, h.Args...)
		cmd.Env = h.Env

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("exec hook: %w", err)
		}
	}

	return nil
}
