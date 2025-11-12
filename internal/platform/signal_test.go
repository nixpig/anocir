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
	}{
		"test signal full name": {
			sigName: "SIGINT",
			signal:  unix.SIGINT,
		},
		"test signal shorthand name": {
			sigName: "TRAP",
			signal:  unix.SIGTRAP,
		},
		"test signal int name": {
			sigName: "9",
			signal:  unix.SIGKILL,
		},
		"test invalid signal name": {
			sigName: "invalid",
			signal:  unix.Signal(0),
		},
		"test empty signal name": {
			sigName: "",
			signal:  unix.Signal(0),
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.signal, ParseSignal(data.sigName))
		})
	}
}
