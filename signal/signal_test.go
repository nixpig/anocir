package signal_test

import (
	"syscall"
	"testing"

	"github.com/nixpig/brownie/signal"
	"github.com/stretchr/testify/assert"
)

func TestFromInt(t *testing.T) {
	sig := signal.FromInt(9)
	assert.Equal(t, syscall.SIGKILL, sig)
}

func TestFromIntInvalid(t *testing.T) {
	sig := signal.FromInt(99)
	assert.Equal(t, syscall.Signal(0), sig)
}

func TestFromStringNumber(t *testing.T) {
	sig := signal.FromString("10")
	assert.Equal(t, syscall.SIGUSR1, sig)
}

func TestFromStringShort(t *testing.T) {
	sig := signal.FromString("CHLD")
	assert.Equal(t, syscall.SIGCHLD, sig)
}

func TestFromStringLong(t *testing.T) {
	sig := signal.FromString("SIGQUIT")
	assert.Equal(t, syscall.SIGQUIT, sig)
}

func TestFromStringInvalid(t *testing.T) {
	sig := signal.FromString("something invalid")
	assert.Equal(t, syscall.Signal(0), sig)
}
