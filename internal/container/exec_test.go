package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckExecutable(t *testing.T) {
	tempDir := t.TempDir()

	nonExePath := filepath.Join(tempDir, "non-executable")
	exePath := filepath.Join(tempDir, "executable")

	err := os.WriteFile(nonExePath, []byte{}, 0o644)
	require.NoError(t, err)

	err = os.WriteFile(exePath, []byte{}, 0o755)
	require.NoError(t, err)

	scenarios := map[string]struct {
		path    string
		wantErr bool
	}{
		"invalid path":     {path: "non-existent", wantErr: true},
		"directory path":   {path: tempDir, wantErr: true},
		"not executable":   {path: nonExePath, wantErr: true},
		"valid executable": {path: exePath, wantErr: false},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			if data.wantErr {
				assert.Error(t, checkExecutable(data.path))
			} else {
				assert.NoError(t, checkExecutable(data.path))
			}
		})
	}
}

func TestSharedNamespace(t *testing.T) {
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
		wantErr           bool
	}{
		"invalid container ns path": {
			containerNSPath:   filepath.Join(tempDir, "invalid"),
			hostNSPath:        isolatedHostNS,
			isSharedNamespace: false,
			wantErr:           true,
		},
		"invalid host ns path": {
			containerNSPath:   isolatedContainerNS,
			hostNSPath:        filepath.Join(tempDir, "invalid"),
			isSharedNamespace: false,
			wantErr:           true,
		},
		"shared namespace paths": {
			containerNSPath:   sharedContainerNS,
			hostNSPath:        sharedHostNS,
			isSharedNamespace: true,
			wantErr:           false,
		},
		"not shared namespace paths": {
			containerNSPath:   isolatedContainerNS,
			hostNSPath:        isolatedHostNS,
			isSharedNamespace: false,
			wantErr:           false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			s, err := sharedNamespace(
				data.containerNSPath,
				data.hostNSPath,
			)
			assert.Equal(t, data.isSharedNamespace, s)

			if data.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
