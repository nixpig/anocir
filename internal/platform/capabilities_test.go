package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/gocapability/capability"
)

func TestResolveCaps(t *testing.T) {
	scenarios := map[string]struct {
		names []string
		caps  []capability.Cap
	}{
		"test single capability": {
			names: []string{"CAP_SYS_ADMIN"},
			caps:  []capability.Cap{capability.CAP_SYS_ADMIN},
		},
		"test multiple capabilities": {
			names: []string{
				"CAP_SETUID",
				"CAP_SETGID",
			},
			caps: []capability.Cap{
				capability.CAP_SETUID,
				capability.CAP_SETGID,
			},
		},
		"test empty capabilities": {
			names: []string{},
			caps:  []capability.Cap{},
		},
		"test invalid capabilities": {
			names: []string{"invalid"},
			caps:  []capability.Cap{},
		},
		"test mixed valid/invalid capabilities": {
			names: []string{
				"CAP_SETUID",
				"invalid",
				"CAP_SETGID",
			},
			caps: []capability.Cap{
				capability.CAP_SETUID,
				capability.CAP_SETGID,
			},
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.caps, resolveCaps(data.names))
		})
	}
}
