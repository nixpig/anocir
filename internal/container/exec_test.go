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
