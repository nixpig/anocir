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

func TestParseEnv(t *testing.T) {
	scenarios := map[string]struct {
		env   []string
		envs  map[string]string
		valid bool
	}{
		"empty env": {
			env:   []string{},
			envs:  map[string]string{},
			valid: true,
		},
		"single env": {
			env:   []string{"foo=bar"},
			envs:  map[string]string{"foo": "bar"},
			valid: true,
		},
		"multiple envs": {
			env:   []string{"foo=bar", "baz=qux"},
			envs:  map[string]string{"foo": "bar", "baz": "qux"},
			valid: true,
		},
		"invalid - missing =": {
			env:   []string{"foobar"},
			envs:  nil,
			valid: false,
		},
		"invalid - too many =": {
			env:   []string{"foo=bar=baz"},
			envs:  nil,
			valid: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			envs, err := parseEnv(data.env)

			assert.Equal(t, data.envs, envs)
			assert.Equal(t, data.valid, err == nil)
		})
	}
}
