package signal_test

import (
	"syscall"
	"testing"

	"github.com/nixpig/brownie/internal/signal"
	"github.com/stretchr/testify/assert"
)

var signalsInt = map[int]syscall.Signal{
	1:  syscall.SIGHUP,
	2:  syscall.SIGINT,
	3:  syscall.SIGQUIT,
	6:  syscall.SIGABRT,
	9:  syscall.SIGKILL,
	15: syscall.SIGTERM,
	17: syscall.SIGCHLD,
	19: syscall.SIGSTOP,
	20: syscall.SIGSTOP,
	21: syscall.SIGSTOP,
	22: syscall.SIGSTOP,
}

var signalsStr = map[string]syscall.Signal{
	"HUP":     syscall.SIGHUP,
	"SIGHUP":  syscall.SIGHUP,
	"INT":     syscall.SIGINT,
	"SIGINT":  syscall.SIGINT,
	"QUIT":    syscall.SIGQUIT,
	"SIGQUIT": syscall.SIGQUIT,
	"ABRT":    syscall.SIGABRT,
	"SIGABRT": syscall.SIGABRT,
	"KILL":    syscall.SIGKILL,
	"SIGKILL": syscall.SIGKILL,
	"TERM":    syscall.SIGTERM,
	"SIGTERM": syscall.SIGTERM,
	"CHLD":    syscall.SIGCHLD,
	"SIGCHLD": syscall.SIGCHLD,
	"STOP":    syscall.SIGSTOP,
	"SIGSTOP": syscall.SIGSTOP,
}

func TestFromInt(t *testing.T) {
	for k, v := range signalsInt {
		sig, err := signal.FromInt(k)
		assert.NoError(t, err)
		assert.Equal(t, v, sig, "")
	}
}

func TestFromString(t *testing.T) {
	for k, v := range signalsStr {
		sig, err := signal.FromString(k)
		assert.NoError(t, err)
		assert.Equal(t, v, sig)
	}
}
