package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestContainerLifecycle(t *testing.T) {
	opts := &Opts{
		ID:      "test-container",
		Bundle:  t.TempDir(),
		Spec:    &specs.Spec{},
		RootDir: t.TempDir(),
		PIDFile: filepath.Join(t.TempDir(), "pid"),
	}

	config, err := json.Marshal(&specs.Spec{})
	assert.NoError(t, err, "Spec should marshal to JSON")

	os.WriteFile(filepath.Join(opts.Bundle, "config.json"), config, 0o644)

	c := New(opts)

	if err := os.MkdirAll(filepath.Join(opts.RootDir, opts.ID), 0o755); err != nil {
		assert.Fail(t, "must create container directory")
	}

	if err := c.Save(); err != nil {
		assert.Fail(t, "Container should save")
	}

	stateFileInfo, err := os.Stat(
		filepath.Join(opts.RootDir, opts.ID, "state.json"),
	)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), stateFileInfo.Mode().Perm())

	exists := Exists(opts.ID, opts.RootDir)
	assert.True(t, exists, "Exists should confirm container exists")

	loaded, err := Load(opts.ID, opts.RootDir)
	assert.NoError(t, err, "Load should load container")

	assert.Equal(t, &Container{
		State: &specs.State{
			Version: "1.3.0",
			ID:      opts.ID,
			Status:  "creating",
			Bundle:  opts.Bundle,
		},
		spec:    opts.Spec,
		RootDir: opts.RootDir,
		containerSock: filepath.Join(
			opts.RootDir,
			opts.ID,
			containerSockFilename,
		),
	}, loaded)
}

func TestRootFS(t *testing.T) {
	scenarios := map[string]struct {
		rootPath, bundlePath, rootFS string
	}{
		"test rootfs with absolute path": {
			rootPath:   "/bin/sh",
			bundlePath: "/bundle",
			rootFS:     "/bin/sh",
		},
		"test rootfs with relative path": {
			rootPath:   "bin/sh",
			bundlePath: "/bundle",
			rootFS:     "/bundle/bin/sh",
		},
		"test rootfs with empty path": {
			rootPath:   "",
			bundlePath: "/bundle",
			rootFS:     "/bundle",
		},
		"test rootfs with dot path": {
			rootPath:   ".",
			bundlePath: "/bundle",
			rootFS:     "/bundle",
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			c := &Container{
				spec:  &specs.Spec{Root: &specs.Root{Path: data.rootPath}},
				State: &specs.State{Bundle: data.bundlePath},
			}

			assert.Equal(t, data.rootFS, c.RootFS())
		})
	}
}

func TestStateFilePath(t *testing.T) {
	opts := &Opts{
		ID:      "test-container",
		Bundle:  t.TempDir(),
		Spec:    &specs.Spec{},
		RootDir: t.TempDir(),
		PIDFile: filepath.Join(t.TempDir(), "pid"),
	}

	c := New(opts)

	assert.Equal(
		t,
		filepath.Join(opts.RootDir, opts.ID, "state.json"),
		c.stateFilepath(),
	)
}

func TestStateChange(t *testing.T) {
	scenarios := map[string]struct {
		state        specs.ContainerState
		canBeStarted bool
		canBeKilled  bool
		canBeDeleted bool
	}{
		"from state creating": {specs.StateCreating, false, false, false},
		"from state created":  {specs.StateCreated, true, true, false},
		"from state running":  {specs.StateRunning, false, true, false},
		"from state stopped":  {specs.StateStopped, false, false, true},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			c := &Container{State: &specs.State{Status: data.state}}

			assert.Equal(
				t,
				data.canBeStarted,
				c.canBeStarted(),
				"container can be started",
			)

			assert.Equal(
				t,
				data.canBeKilled,
				c.canBeKilled(),
				"container can be killed",
			)

			assert.Equal(
				t,
				data.canBeDeleted,
				c.canBeDeleted(),
				"container can be deleted",
			)
		})
	}
}
