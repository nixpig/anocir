// internal/operations/create.go

package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nixpig/anocir/internal/container"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type CreateOpts struct {
	ID     string
	Bundle string
}

func Create(opts *CreateOpts) error {
	bundle, err := filepath.Abs(opts.Bundle)
	if err != nil {
		return fmt.Errorf("absolute path from bundle: %w", err)
	}

	config, err := os.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var spec *specs.Spec
	if err := json.Unmarshal(config, spec); err != nil {
		return fmt.Errorf("unmarshall config: %w", err)
	}

	if _, err := container.New(&container.NewContainerOpts{
		ID:     opts.ID,
		Bundle: bundle,
		Spec:   spec,
	}); err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	return nil
}
