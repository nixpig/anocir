package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opencontainers/runtime-spec/specs-go"
)

type Bundle struct {
	Path     string
	Rootfs   string
	SpecPath string
	Spec     specs.Spec
}

func New(path string) (*Bundle, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute path to bundle: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("check bundle path: %w", err)
	}

	specPath := filepath.Join(absPath, "config.json")
	specJSON, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec from bundle: %w", err)
	}

	var spec specs.Spec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	rootfs := filepath.Join(path, spec.Root.Path)
	if _, err := os.Stat(rootfs); err != nil {
		return nil, fmt.Errorf("check rootfs path: %w", err)
	}

	return &Bundle{
		Path:     absPath,
		Spec:     spec,
		SpecPath: specPath,
		Rootfs:   rootfs,
	}, nil
}
