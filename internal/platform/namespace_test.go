package platform_test

import (
	"os"
	"syscall"
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
				Path: "/proc/self/ns/pid",
			},
			err: platform.ErrInvalidNamespacePath,
		},
		"invalid pid namespace path": {
			ns: &specs.LinuxNamespace{
				Type: specs.PIDNamespace,
				Path: "/proc/self/ns/net",
			},
			err: platform.ErrInvalidNamespacePath,
		},
		"empty namespace": {
			ns:  &specs.LinuxNamespace{},
			err: nil,
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			err := platform.ValidateNSPath(data.ns)
			assert.ErrorIs(t, err, data.err)
		})
	}
}

func TestBuildUserNSMappings(t *testing.T) {
	scenarios := map[string]struct {
		specUIDMappings     []specs.LinuxIDMapping
		specGIDMappings     []specs.LinuxIDMapping
		expectedUIDMappings []syscall.SysProcIDMap
		expectedGIDMappings []syscall.SysProcIDMap
	}{
		"empty uid and gid mappings": {
			specUIDMappings: []specs.LinuxIDMapping{},
			specGIDMappings: []specs.LinuxIDMapping{},
			expectedUIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getuid(),
					Size:        1,
				},
			},
			expectedGIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getgid(),
					Size:        1,
				},
			},
		},
		"missing uid and gid mappings": {
			specUIDMappings: nil,
			specGIDMappings: nil,
			expectedUIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getuid(),
					Size:        1,
				},
			},
			expectedGIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getgid(),
					Size:        1,
				},
			},
		},
		"only uid mappings": {
			specUIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
			specGIDMappings: []specs.LinuxIDMapping{},
			expectedUIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
			expectedGIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getgid(),
					Size:        1,
				},
			},
		},
		"only gid mappings": {
			specUIDMappings: []specs.LinuxIDMapping{},
			specGIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
			expectedUIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getuid(),
					Size:        1,
				},
			},
			expectedGIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
		},
		"single uid and gid mapping": {
			specUIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
			specGIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
			expectedUIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
			expectedGIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 1000,
					HostID:      1000,
					Size:        1,
				},
			},
		},
		"multiple uid and gid mappings": {
			specUIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: 0,
					HostID:      1000,
					Size:        1,
				},
				{
					ContainerID: 1,
					HostID:      100000,
					Size:        65536,
				},
			},
			specGIDMappings: []specs.LinuxIDMapping{
				{
					ContainerID: 0,
					HostID:      1000,
					Size:        1,
				},
				{
					ContainerID: 1,
					HostID:      100000,
					Size:        65536,
				},
			},
			expectedUIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      1000,
					Size:        1,
				},
				{
					ContainerID: 1,
					HostID:      100000,
					Size:        65536,
				},
			},
			expectedGIDMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      1000,
					Size:        1,
				},
				{
					ContainerID: 1,
					HostID:      100000,
					Size:        65536,
				},
			},
		},
	}

	for scenario, data := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			uidMappings, gidMappings := platform.BuildUserNSMappings(
				data.specUIDMappings,
				data.specGIDMappings,
			)

			assert.Equal(t, data.expectedUIDMappings, uidMappings)
			assert.Equal(t, data.expectedGIDMappings, gidMappings)
		})
	}
}
