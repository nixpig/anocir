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

func TestBuildSystemdCgroupPath(t *testing.T) {
	scenarios := map[string]struct {
		cgroupsPath       string
		containerID       string
		systemdCgroupPath string
	}{
		"test cgroupsPath without slice suffix": {
			cgroupsPath:       "/etc/systemd/system/user-1000",
			containerID:       "",
			systemdCgroupPath: "/etc/systemd/system/user-1000.slice",
		},
		"test cgroupsPath with slice suffix": {
			cgroupsPath:       "/etc/systemd/system/user-1000.slice",
			containerID:       "",
			systemdCgroupPath: "/etc/systemd/system/user-1000.slice",
		},
		"test empty cgroupsPath and valid containerID": {
			cgroupsPath:       "",
			containerID:       "test-container",
			systemdCgroupPath: "test-container.slice",
		},
		"test empty cgroupsPath and empty containerID": {
			cgroupsPath:       "",
			containerID:       "",
			systemdCgroupPath: "",
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			assert.Equal(
				t,
				data.systemdCgroupPath,
				buildSystemdCGroupPath(data.cgroupsPath, data.containerID),
			)
		})
	}
}
