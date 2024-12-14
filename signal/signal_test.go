package signal_test

import (
	"syscall"
	"testing"

	"github.com/nixpig/brownie/signal"
	"github.com/stretchr/testify/assert"
)

func TestFromInt(t *testing.T) {
	sig, err := signal.FromInt(9)
	assert.Equal(t, syscall.SIGKILL, sig)
	assert.NoError(t, err)
}

func TestFromIntInvalid(t *testing.T) {
	sig, err := signal.FromInt(99)
	assert.Equal(t, syscall.Signal(0), sig)
	assert.Error(t, err)
}

func TestFromStringNumber(t *testing.T) {
	sig, err := signal.FromString("10")
	assert.Equal(t, syscall.SIGUSR1, sig)
	assert.NoError(t, err)
}

func TestFromStringShort(t *testing.T) {
	sig, err := signal.FromString("CHLD")
	assert.Equal(t, syscall.SIGCHLD, sig)
	assert.NoError(t, err)
}

func TestFromStringLong(t *testing.T) {
	sig, err := signal.FromString("SIGQUIT")
	assert.Equal(t, syscall.SIGQUIT, sig)
	assert.NoError(t, err)
}

func TestFromStringInvalid(t *testing.T) {
	sig, err := signal.FromString("something invalid")
	assert.Equal(t, syscall.Signal(0), sig)
	assert.Error(t, err)
}
