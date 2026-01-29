package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestParseSignal(t *testing.T) {
	scenarios := map[string]struct {
		sigName string
		signal  unix.Signal
		err     error
	}{
		"test signal full name": {
			sigName: "SIGINT",
			signal:  unix.SIGINT,
			err:     nil,
		},
		"test signal shorthand name": {
			sigName: "TRAP",
			signal:  unix.SIGTRAP,
			err:     nil,
		},
		"test signal int name": {
			sigName: "9",
			signal:  unix.SIGKILL,
			err:     nil,
		},
		"test invalid signal name": {
			sigName: "invalid",
			signal:  unix.Signal(0),
			err:     ErrUnknownSignal,
		},
		"test empty signal name": {
			sigName: "",
			signal:  unix.Signal(0),
			err:     ErrUnknownSignal,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			sig, err := ParseSignal(data.sigName)
			assert.Equal(t, data.signal, sig)
			assert.Equal(t, err, data.err)
		})
	}
}
