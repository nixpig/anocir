package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveCaps(t *testing.T) {
	scenarios := map[string]struct {
		names []string
		caps  []Cap
	}{
		"test single capability": {
			names: []string{"CAP_SYS_ADMIN"},
			caps:  []Cap{CAP_SYS_ADMIN},
		},
		"test multiple capabilities": {
			names: []string{
				"CAP_SETUID",
				"CAP_SETGID",
			},
			caps: []Cap{
				CAP_SETUID,
				CAP_SETGID,
			},
		},
		"test empty capabilities": {
			names: []string{},
			caps:  []Cap{},
		},
		"test invalid capabilities": {
			names: []string{"invalid"},
			caps:  []Cap{},
		},
		"test mixed valid/invalid capabilities": {
			names: []string{
				"CAP_SETUID",
				"invalid",
				"CAP_SETGID",
			},
			caps: []Cap{
				CAP_SETUID,
				CAP_SETGID,
			},
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.caps, resolveCaps(data.names))
		})
	}
}
