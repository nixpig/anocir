package container

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

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
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			c := &Container{
				Spec:  &specs.Spec{Root: &specs.Root{Path: data.rootPath}},
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
