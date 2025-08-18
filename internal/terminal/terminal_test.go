package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPty(t *testing.T) {
	pty, err := NewPty()

	assert.NoError(t, err)
	assert.NotNil(t, pty)
	assert.NotNil(t, pty.Master)
	assert.NotNil(t, pty.Slave)
}
