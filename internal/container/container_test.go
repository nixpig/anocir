package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerLifecycle(t *testing.T) {
	c := newTestContainer(t, &specs.Spec{Linux: &specs.Linux{}})

	stateFileInfo, err := os.Stat(
		filepath.Join(c.RootDir, c.State.ID, "state.json"),
	)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), stateFileInfo.Mode().Perm())

	exists := Exists(c.State.ID, c.RootDir)
	assert.True(t, exists, "Exists should confirm container exists")

	loaded, err := Load(c.State.ID, c.RootDir)
	assert.NoError(t, err, "Load should load container")

	assert.Equal(t, &Container{
		State: &specs.State{
			Version: "1.3.0",
			ID:      c.State.ID,
			Status:  "creating",
			Bundle:  c.State.Bundle,
		},
		spec:          c.GetSpec(),
		RootDir:       c.RootDir,
		containerSock: containerSockPath(c.State.Bundle),
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

			assert.Equal(t, data.rootFS, c.rootFS())
		})
	}
}

func TestStateChange(t *testing.T) {
	scenarios := map[string]struct {
		state        specs.ContainerState
		canBeStarted bool
		canBeKilled  bool
		canBeDeleted bool
		canBePaused  bool
		canBeResumed bool
	}{
		"from state creating": {specs.StateCreating, false, false, false, false, false},
		"from state created":  {specs.StateCreated, true, true, false, false, false},
		"from state running":  {specs.StateRunning, false, true, false, true, false},
		"from state stopped":  {specs.StateStopped, false, false, true, false, false},
		"from state paused":   {specs.ContainerState("paused"), false, false, false, false, true},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			c := &Container{State: &specs.State{Status: data.state}}

			assert.Equal(t, data.canBeStarted, c.canBeStarted(), "container can be started")
			assert.Equal(t, data.canBeKilled, c.canBeKilled(), "container can be killed")
			assert.Equal(t, data.canBeDeleted, c.canBeDeleted(), "container can be deleted")
			assert.Equal(t, data.canBePaused, c.canBePaused(), "container can be paused")
			assert.Equal(t, data.canBeResumed, c.canBeResumed(), "container can be resumed")
		})
	}
}

func TestHasMountNamespace(t *testing.T) {
	scenarios := map[string]struct {
		hasMountNamespace bool
	}{
		"has mount namespace":           {hasMountNamespace: true},
		"does not have mount namespace": {hasMountNamespace: false},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			c := &Container{
				State: &specs.State{},
				spec: &specs.Spec{
					Linux: &specs.Linux{
						Namespaces: []specs.LinuxNamespace{
							{Type: specs.PIDNamespace},
							{Type: specs.IPCNamespace},
						},
					},
				},
			}

			if data.hasMountNamespace {
				c.spec.Linux.Namespaces = append(c.spec.Linux.Namespaces, specs.LinuxNamespace{Type: specs.MountNamespace})
			}

			assert.Equal(t, data.hasMountNamespace, c.hasMountNamespace())
		})
	}
}

func TestReloadState(t *testing.T) {
	c := newTestContainer(t, &specs.Spec{Linux: &specs.Linux{}})

	require.Equal(t, c.State.Status, specs.StateCreating)

	state, err := c.GetState()
	require.NoError(t, err)

	state.Status = specs.StateStopped

	stateFile, err := os.OpenFile(c.stateFilepath(), os.O_WRONLY, 0o644)
	require.NoError(t, err)

	err = json.NewEncoder(stateFile).Encode(state)
	require.NoError(t, err)

	c.reloadState()
	assert.Equal(t, c.State.Status, specs.StateStopped)
}

func newTestContainer(t *testing.T, spec *specs.Spec) *Container {
	t.Helper()

	opts := &Opts{
		ID:      "test-container",
		Bundle:  t.TempDir(),
		Spec:    spec,
		RootDir: t.TempDir(),
		PIDFile: filepath.Join(t.TempDir(), "pid"),
	}

	config, err := json.Marshal(&specs.Spec{Linux: &specs.Linux{}})
	require.NoError(t, err, "Spec should marshal to JSON")

	err = os.WriteFile(filepath.Join(opts.Bundle, "config.json"), config, 0o644)
	require.NoError(t, err)

	c, err := New(opts)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(opts.RootDir, opts.ID), 0o755)
	require.NoError(t, err)

	err = c.Save()
	require.NoError(t, err)

	return c
}
