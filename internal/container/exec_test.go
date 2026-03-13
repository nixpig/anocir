package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckExecutable(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	nonExePath := filepath.Join(tempDir, "non-executable")
	exePath := filepath.Join(tempDir, "executable")

	err := os.WriteFile(nonExePath, []byte{}, 0o644)
	require.NoError(t, err)

	err = os.WriteFile(exePath, []byte{}, 0o755)
	require.NoError(t, err)

	scenarios := map[string]struct {
		path      string
		assertErr assert.ErrorAssertionFunc
	}{
		"invalid path":     {path: "non-existent", assertErr: assert.Error},
		"directory path":   {path: tempDir, assertErr: assert.Error},
		"not executable":   {path: nonExePath, assertErr: assert.Error},
		"valid executable": {path: exePath, assertErr: assert.NoError},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			data.assertErr(t, checkExecutable(data.path))
		})
	}
}

func TestSharedNamespace(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	isolatedContainerNS := filepath.Join(tempDir, "isolated_container_ns")
	err := os.WriteFile(isolatedContainerNS, []byte{}, 0o644)
	require.NoError(t, err)

	isolatedHostNS := filepath.Join(tempDir, "isolated_host_ns")
	err = os.WriteFile(isolatedHostNS, []byte{}, 0o644)
	require.NoError(t, err)

	sharedContainerNS := filepath.Join(tempDir, "shared_container_ns")
	err = os.WriteFile(sharedContainerNS, []byte{}, 0o644)
	require.NoError(t, err)

	sharedHostNS := filepath.Join(tempDir, "shared_host_ns")
	err = os.Link(sharedContainerNS, sharedHostNS)
	require.NoError(t, err)

	scenarios := map[string]struct {
		containerNSPath   string
		hostNSPath        string
		isSharedNamespace bool
		assertErr         assert.ErrorAssertionFunc
	}{
		"invalid container ns path": {
			containerNSPath:   filepath.Join(tempDir, "invalid"),
			hostNSPath:        isolatedHostNS,
			isSharedNamespace: false,
			assertErr:         assert.Error,
		},
		"invalid host ns path": {
			containerNSPath:   isolatedContainerNS,
			hostNSPath:        filepath.Join(tempDir, "invalid"),
			isSharedNamespace: false,
			assertErr:         assert.Error,
		},
		"shared namespace paths": {
			containerNSPath:   sharedContainerNS,
			hostNSPath:        sharedHostNS,
			isSharedNamespace: true,
			assertErr:         assert.NoError,
		},
		"not shared namespace paths": {
			containerNSPath:   isolatedContainerNS,
			hostNSPath:        isolatedHostNS,
			isSharedNamespace: false,
			assertErr:         assert.NoError,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			s, err := sharedNamespace(
				data.containerNSPath,
				data.hostNSPath,
			)
			assert.Equal(t, data.isSharedNamespace, s)
			data.assertErr(t, err)
		})
	}
}
