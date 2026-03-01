package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	rootfs := filepath.Join(tempDir, "rootfs")

	require.NoError(t, os.Mkdir(rootfs, 0o755))

	scenarios := map[string]struct {
		symlinks map[string]string
		rootfs   string
		wantErr  bool
	}{
		"invalid rootfs": {
			symlinks: map[string]string{},
			rootfs:   filepath.Join(tempDir, "invalid"),
			wantErr:  true,
		},
		"valid rootfs and symlinks": {
			symlinks: map[string]string{
				"/proc/self/fd/0": "dev/stdin",
				"/proc/self/fd/1": "dev/stdout",
				"/proc/self/fd/2": "dev/stderr",
			},
			rootfs:  rootfs,
			wantErr: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			for _, dest := range data.symlinks {
				d := filepath.Dir(filepath.Join(data.rootfs, dest))
				if d != "" {
					require.NoError(t, os.MkdirAll(d, 0o755))
				}
			}

			err := createSymlinks(data.symlinks, data.rootfs)

			if data.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				for _, dest := range data.symlinks {
					_, err := os.Lstat(filepath.Join(rootfs, dest))
					require.NoError(t, err)
				}
			}
		})
	}
}
