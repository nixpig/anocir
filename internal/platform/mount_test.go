package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestMountValidatePropagationFlag(t *testing.T) {
	scenarios := map[string]struct {
		flag  uintptr
		valid bool
	}{
		"zero flag is valid":     {flag: 0, valid: true},
		"MS_SHARED is valid":     {flag: unix.MS_SHARED, valid: true},
		"MS_PRIVATE is valid":    {flag: unix.MS_PRIVATE, valid: true},
		"MS_SLAVE is valid":      {flag: unix.MS_SLAVE, valid: true},
		"MS_UNBINDABLE is valid": {flag: unix.MS_UNBINDABLE, valid: true},
		"MS_SHARED|MS_REC is valid": {
			flag:  unix.MS_SHARED | unix.MS_REC,
			valid: true,
		},
		"MS_PRIVATE|MS_REC is valid": {
			flag:  unix.MS_PRIVATE | unix.MS_REC,
			valid: true,
		},
		"MS_SLAVE|MS_REC is valid": {
			flag:  unix.MS_SLAVE | unix.MS_REC,
			valid: true,
		},
		"MS_UNBINDABLE|MS_REC is valid": {
			flag:  unix.MS_UNBINDABLE | unix.MS_REC,
			valid: true,
		},
		"MS_BIND is invalid": {
			flag:  unix.MS_BIND,
			valid: false,
		},
		"MS_RDONLY is invalid": {
			flag:  unix.MS_RDONLY,
			valid: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.valid, validatePropagationFlag(data.flag))
		})
	}
}
