package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSysctlPath(t *testing.T) {
	scenarios := map[string]struct {
		sysctl string
		path   string
	}{
		"test valid sysctl": {
			sysctl: "vm.swappiness",
			path:   "/proc/sys/vm/swappiness",
		},
		"test empty sysctl": {
			sysctl: "",
			path:   "/proc/sys",
		},
		"test parent traversal": {
			sysctl: "foo...bar",
			path:   "/proc/sys/foo/bar",
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.path, sysctlPath(data.sysctl))
		})
	}
}
