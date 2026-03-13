package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSymlinks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	rootfs := filepath.Join(tempDir, "rootfs")

	require.NoError(t, os.Mkdir(rootfs, 0o755))

	scenarios := map[string]struct {
		symlinks  map[string]string
		rootfs    string
		assertErr assert.ErrorAssertionFunc
	}{
		"invalid rootfs": {
			symlinks:  map[string]string{},
			rootfs:    filepath.Join(tempDir, "invalid"),
			assertErr: assert.Error,
		},
		"valid rootfs and symlinks": {
			symlinks: map[string]string{
				"/proc/self/fd/0": "dev/stdin",
				"/proc/self/fd/1": "dev/stdout",
				"/proc/self/fd/2": "dev/stderr",
			},
			rootfs:    rootfs,
			assertErr: assert.NoError,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			for _, dest := range data.symlinks {
				d := filepath.Dir(filepath.Join(data.rootfs, dest))
				if d != "" {
					require.NoError(t, os.MkdirAll(d, 0o755))
				}
			}

			err := createSymlinks(data.symlinks, data.rootfs)
			data.assertErr(t, err)
			for _, dest := range data.symlinks {
				_, err := os.Lstat(filepath.Join(rootfs, dest))
				assert.NoError(t, err)
			}
		})
	}
}
