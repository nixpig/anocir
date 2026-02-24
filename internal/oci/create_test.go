package oci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContainerSpec(t *testing.T) {
	tempDir := t.TempDir()
	spec := &specs.Spec{Version: "1.0.0", Hostname: "tester"}

	specFile, err := os.Create(filepath.Join(tempDir, "config.json"))
	require.NoError(t, err)

	err = json.NewEncoder(specFile).Encode(spec)
	require.NoError(t, err)

	s, err := getContainerSpec(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, spec, s)
}

func TestCreateContainerDirs(t *testing.T) {
	tempDir := t.TempDir()
	rootDir := tempDir + "root"
	containerID := "12345"

	err := createContainerDirs(rootDir, containerID)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(rootDir, containerID))
	assert.NoError(t, err)
	assert.NotNil(t, info)
}
