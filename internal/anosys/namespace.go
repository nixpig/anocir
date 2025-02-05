package anosys

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var NamespaceFlags = map[specs.LinuxNamespaceType]uintptr{
	specs.PIDNamespace:     unix.CLONE_NEWPID,
	specs.NetworkNamespace: unix.CLONE_NEWNET,
	specs.MountNamespace:   unix.CLONE_NEWNS,
	specs.IPCNamespace:     unix.CLONE_NEWIPC,
	specs.UTSNamespace:     unix.CLONE_NEWUTS,
	specs.UserNamespace:    unix.CLONE_NEWUSER,
	specs.CgroupNamespace:  unix.CLONE_NEWCGROUP,
	specs.TimeNamespace:    unix.CLONE_NEWTIME,
}

var NamespaceEnvs = map[specs.LinuxNamespaceType]string{
	specs.PIDNamespace:     "pid",
	specs.NetworkNamespace: "net",
	specs.MountNamespace:   "mnt",
	specs.IPCNamespace:     "ipc",
	specs.UTSNamespace:     "uts",
	specs.UserNamespace:    "user",
	specs.CgroupNamespace:  "cgroup",
	specs.TimeNamespace:    "time",
}
