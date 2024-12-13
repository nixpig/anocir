package user

import (
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func BuildUidMappings(specUIDMappings []specs.LinuxIDMapping) []syscall.SysProcIDMap {
	uidMappings := make([]syscall.SysProcIDMap, len(specUIDMappings))

	for i, m := range specUIDMappings {
		uidMappings[i] = syscall.SysProcIDMap{
			ContainerID: int(m.ContainerID),
			HostID:      int(m.HostID),
			Size:        int(m.Size),
		}
	}

	return uidMappings
}

func BuildGidMappings(specGIDMappings []specs.LinuxIDMapping) []syscall.SysProcIDMap {
	gidMappings := make([]syscall.SysProcIDMap, len(specGIDMappings))

	for i, g := range specGIDMappings {
		gidMappings[i] = syscall.SysProcIDMap{
			ContainerID: int(g.ContainerID),
			HostID:      int(g.HostID),
			Size:        int(g.Size),
		}
	}

	return gidMappings
}
