package signal_test

import (
	"testing"

	"github.com/nixpig/brownie/signal"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestFromStringNumber(t *testing.T) {
	sig := signal.FromString("10")
	assert.Equal(t, unix.SIGUSR1, sig)
}

func TestFromStringShort(t *testing.T) {
	sig := signal.FromString("CHLD")
	assert.Equal(t, unix.SIGCHLD, sig)
}

func TestFromStringLong(t *testing.T) {
	sig := signal.FromString("SIGQUIT")
	assert.Equal(t, unix.SIGQUIT, sig)
}

func TestFromStringInvalid(t *testing.T) {
	sig := signal.FromString("something invalid")
	assert.Equal(t, unix.Signal(0), sig)
}
