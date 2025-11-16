package platform_test

import (
	"testing"

	"github.com/nixpig/anocir/internal/platform"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceMappings(t *testing.T) {
	assert.Len(t, platform.NamespaceEnvs, 8)
	assert.Len(t, platform.NamespaceFlags, 8)

	for key := range platform.NamespaceFlags {
		_, ok := platform.NamespaceEnvs[key]

		assert.True(t, ok, "missing namespace env for '%s'", key)
	}
}

func TestValidateNSPath(t *testing.T) {
	scenarios := map[string]struct {
		ns  *specs.LinuxNamespace
		err error
	}{
		"net namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.NetworkNamespace,
				Path: "/proc/self/ns/net",
			},
			err: nil,
		},
		"mnt namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.MountNamespace,
				Path: "/proc/self/ns/mnt",
			},
			err: nil,
		},
		"ipc namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.IPCNamespace,
				Path: "/proc/self/ns/ipc",
			},
			err: nil,
		},
		"uts namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.UTSNamespace,
				Path: "/proc/self/ns/uts",
			},
			err: nil,
		},
		"user namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.UserNamespace,
				Path: "/proc/self/ns/user",
			},
			err: nil,
		},
		"cgroup namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.CgroupNamespace,
				Path: "/proc/self/ns/cgroup",
			},
			err: nil,
		},
		"time namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.TimeNamespace,
				Path: "/proc/self/ns/time",
			},
			err: nil,
		},
		"pid (pid) namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.PIDNamespace,
				Path: "/proc/self/ns/pid",
			},
			err: nil,
		},
		"pid (pid_for_children) namespace": {
			ns: &specs.LinuxNamespace{
				Type: specs.PIDNamespace,
				Path: "/proc/self/ns/pid_for_children",
			},
			err: nil,
		},

		"invalid namespace type": {
			ns: &specs.LinuxNamespace{
				Type: specs.LinuxNamespaceType("invalid"),
				Path: "/proc/self/ns/net",
			},
			err: platform.ErrInvalidNamespacePath,
		},
		"invalid namespace path": {
			ns: &specs.LinuxNamespace{
				Type: specs.NetworkNamespace,
				Path: "/proc/self/ns/invalid",
			},
			err: platform.ErrInvalidNamespacePath,
		},
		"invalid pid namespace path": {
			ns: &specs.LinuxNamespace{
				Type: specs.PIDNamespace,
				Path: "/proc/self/ns/invalid",
			},
			err: platform.ErrInvalidNamespacePath,
		},
		"empty namespace": {
			ns:  &specs.LinuxNamespace{},
			err: platform.ErrInvalidNamespacePath,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			err := platform.ValidateNSPath(data.ns)
			assert.ErrorIs(t, err, data.err)
		})
	}
}
