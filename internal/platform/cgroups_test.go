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

func TestBuildSystemdCGroupSliceAndGroup(t *testing.T) {
	scenarios := map[string]struct {
		cgroupsPath        string
		containerID        string
		systemdCgroupSlice string
		systemdCgroupScope string
	}{
		"test empty containerID and cgroupsPath without slice suffix": {
			cgroupsPath:        "/etc/systemd/system/user-1000",
			containerID:        "",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "",
		},
		"test empty containerID and cgroupsPath with slice suffix": {
			cgroupsPath:        "/etc/systemd/system/user-1000.slice",
			containerID:        "",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "",
		},
		"test empty containerID and empty cgroupsPath": {
			cgroupsPath:        "",
			containerID:        "",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "",
		},
		"test valid containerID and cgroupsPath without slice suffix": {
			cgroupsPath:        "/etc/systemd/system/user-1000",
			containerID:        "test-container",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test valid containerID and cgroupsPath with slice suffix": {
			cgroupsPath:        "/etc/systemd/system/user-1000.slice",
			containerID:        "test-container",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test valid containerID and empty cgroupsPath": {
			cgroupsPath:        "",
			containerID:        "test-container",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test colon format with explicit slice prefix and name": {
			cgroupsPath:        "system.slice:anocir:test-container",
			containerID:        "ignored-container",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test colon format with empty slice defaults to system.slice": {
			cgroupsPath:        ":anocir:test-container",
			containerID:        "ignored-container",
			systemdCgroupSlice: "system.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test colon format with dash slice defaults to root": {
			cgroupsPath:        "-:anocir:test-container",
			containerID:        "ignored-container",
			systemdCgroupSlice: "/",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test colon format with custom slice": {
			cgroupsPath:        "user.slice:anocir:test-container",
			containerID:        "ignored-container",
			systemdCgroupSlice: "user.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test colon format with empty prefix": {
			cgroupsPath:        "machine.slice::test-container",
			containerID:        "ignored-container",
			systemdCgroupSlice: "machine.slice",
			systemdCgroupScope: "test-container.scope",
		},
		"test colon format with name ending in slice suffix": {
			cgroupsPath:        "custom.slice:anocir:test-container.slice",
			containerID:        "ignored-container",
			systemdCgroupSlice: "custom.slice",
			systemdCgroupScope: "test-container.slice",
		},
		"test colon format with slice and prefix and empty name": {
			cgroupsPath:        "machine.slice:anocir:",
			containerID:        "test-container",
			systemdCgroupSlice: "machine.slice",
			systemdCgroupScope: "anocir-test-container.scope",
		},
		"test colon format with slice and empty prefix and empty name": {
			cgroupsPath:        "machine.slice::",
			containerID:        "test-container",
			systemdCgroupSlice: "machine.slice",
			systemdCgroupScope: "test-container.scope",
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			slice, scope := buildSystemdCGroupSliceAndGroup(
				data.cgroupsPath,
				data.containerID,
			)
			assert.Equal(t, data.systemdCgroupSlice, slice)
			assert.Equal(t, data.systemdCgroupScope, scope)
		})
	}
}
