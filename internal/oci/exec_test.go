package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUser(t *testing.T) {
	scenarios := map[string]struct {
		user  string
		uid   int
		gid   int
		valid bool
	}{
		"empty user": {
			user:  "",
			uid:   0,
			gid:   0,
			valid: true,
		},
		"uid only": {
			user:  "1000",
			uid:   1000,
			gid:   0,
			valid: true,
		},
		"uid and gid": {
			user:  "1000:1001",
			uid:   1000,
			gid:   1001,
			valid: true,
		},
		"missing uid": {
			user:  ":1001",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"missing gid": {
			user:  "1000:",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"invalid uid only": {
			user:  "invalid",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"invalid uid, valid gid": {
			user:  "invalid:1001",
			uid:   0,
			gid:   0,
			valid: false,
		},
		"valid uid, invalid gid": {
			user:  "1000:invalid",
			uid:   0,
			gid:   0,
			valid: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			uid, gid, err := parseUser(data.user)

			assert.Equal(t, data.uid, uid)
			assert.Equal(t, data.gid, gid)
			assert.Equal(t, data.valid, err == nil)
		})
	}
}

func TestParseProcessFile(t *testing.T) {
	// TODO: Add tests.
}

func TestParseProcessFlags(t *testing.T) {
	// TODO: Add tests.
}
