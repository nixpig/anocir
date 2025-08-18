package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateCmd(t *testing.T) {
	cmd := createCmd()

	assert.Equal(t, "create [flags] CONTAINER_ID", cmd.Use)

	bundleFlag := cmd.Flag("bundle")
	assert.NotNil(t, bundleFlag)
	assert.Equal(t, "b", bundleFlag.Shorthand)

	consoleSocketFlag := cmd.Flag("console-socket")
	assert.NotNil(t, consoleSocketFlag)
	assert.Equal(t, "s", consoleSocketFlag.Shorthand)

	pidFileFlag := cmd.Flag("pid-file")
	assert.NotNil(t, pidFileFlag)
	assert.Equal(t, "p", pidFileFlag.Shorthand)
}
