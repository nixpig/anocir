package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCgroupPath(t *testing.T) {
	scenarios := map[string]struct {
		path  string
		valid bool
	}{
		"test absolute path": {
			path:  "/foo/bar/baz",
			valid: true,
		},
		"test relative path": {
			path:  "foo/bar/baz",
			valid: true,
		},
		"test valid parent traversal": {
			path:  "foo/bar/../baz",
			valid: true,
		},
		"test invalid parent traversal": {
			path:  "foo/../../bar/baz",
			valid: false,
		},
		"test empty path": {
			path:  "",
			valid: false,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(t, data.valid, validateCgroupPath(data.path))
		})
	}
}
