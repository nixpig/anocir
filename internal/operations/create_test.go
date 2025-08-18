package operations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nixpig/anocir/internal/container"
	"github.com/stretchr/testify/assert"
)

func TestCreate_ContainerExists(t *testing.T) {
	tempDir := t.TempDir()
	container.SetContainerRootDir(tempDir)

	containerID := "test-container"
	containerDir := filepath.Join(tempDir, containerID)
	err := os.MkdirAll(containerDir, 0755)
	assert.NoError(t, err)

	opts := &CreateOpts{
		ID: containerID,
	}

	err = Create(opts)
	assert.Error(t, err)
}
