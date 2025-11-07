package container

import (
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestRootFS(t *testing.T) {
	scenarios := map[string]struct{ rootPath, bundlePath, rootFS string }{
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
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			c := &Container{
				Spec:  &specs.Spec{Root: &specs.Root{Path: data.rootPath}},
				State: &specs.State{Bundle: data.bundlePath},
			}

			assert.Equal(t, c.rootFS(), data.rootFS)
		})
	}
}
