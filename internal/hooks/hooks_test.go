package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestExecHooks(t *testing.T) {
	tempDir := t.TempDir()
	hookPath := filepath.Join(tempDir, "hook.sh")
	outputFile := filepath.Join(tempDir, "hook_was_called")

	// hook script
	script := `#!/bin/sh
echo "hook called" > ` + outputFile + `
`
	err := os.WriteFile(hookPath, []byte(script), 0o755)
	assert.NoError(t, err)

	hooks := []specs.Hook{
		{
			Path: hookPath,
		},
	}

	state := &specs.State{
		ID: "test-container",
	}

	err = ExecHooks(hooks, state)
	assert.NoError(t, err)

	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "hook was not called")
}

func TestExecHooks_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	hookPath := filepath.Join(tempDir, "hook.sh")
	hookTimeout := 1

	// hook script that sleeps for longer than the timeout
	script := `#!/bin/sh
sleep 2
`
	err := os.WriteFile(hookPath, []byte(script), 0o755)
	assert.NoError(t, err)

	hooks := []specs.Hook{
		{
			Path:    hookPath,
			Timeout: &hookTimeout,
		},
	}

	state := &specs.State{
		ID: "test-container",
	}

	err = ExecHooks(hooks, state)
	assert.Error(t, err)
}
