package config

import "github.com/nixpig/brownie/pkg"

type NamespaceType string
type DeviceType string

const (
	pid     NamespaceType = "pid"
	network NamespaceType = "network"
	mount   NamespaceType = "mount"
	ipc     NamespaceType = "ipc"
	uts     NamespaceType = "uts"
	user    NamespaceType = "user"
	cgroup  NamespaceType = "cgroup"
	time    NamespaceType = "time"

	allDevices           DeviceType = "a"
	blockDevice          DeviceType = "b"
	charDevice           DeviceType = "c"
	unbufferedCharDevice DeviceType = "u"
	fifoDevice           DeviceType = "p"
)

type ContainerProcessState struct {
	OCIVersion string    `json:"ociVersion"`
	FDs        []string  `json:"fds,omitempty"`
	PID        int       `json:"pid"`
	Metadata   string    `json:"metadata,omitempty"`
	State      pkg.State `json:"state"`
}

type Config struct {
	OCIVersion string      `json:"ociVersion"`
	Root       Root        `json:"root,omitempty"`
	Mounts     []Mount     `json:"mounts,omitempty"`
	Process    Process     `json:"process,omitempty"`
	Hostname   string      `json:"hostname,omitempty"`
	DomainName string      `json:"domainname,omitempty"`
	Linux      LinuxConfig `json:"linux"`
	Hooks      Hooks       `json:"hooks,omitempty"`
}

type Root struct {
	Path     string `json:"path"`
	Readonly *bool  `json:"readonly,omitempty"`
}

// https://github.com/opencontainers/runtime-spec/blob/main/config.md#linux-mount-options
type Mount struct {
	Destination string       `json:"destination"`
	Source      string       `json:"source,omitempty"`
	Options     []string     `json:"options,omitempty"`
	Type        string       `json:"type"`
	UIDMappings []UIDMapping `json:"uidMappings,omitempty"`
	GIDMappings []UIDMapping `json:"gidMappings,omitempty"`
}

type Process struct {
	//omitting commandLine attribute, since it's Window's specific

	User        User                        `json:"user"`
	Terminal    *bool                       `json:"terminal,omitempty"`
	ConsoleSize struct{ Height, Width int } `json:"consoleSize,omitempty"`
	CWD         string                      `json:"cwd"`
	Env         []string                    `json:"env,omitempty"`
	Args        []string                    `json:"args,omitempty"`
	RLimits     []RLimit                    `json:"rlimits,omitempty"`

	// linux-specific properties
	AppArmorProfile string          `json:"apparmorProfile,omitempty"`
	Capabilities    []Capability    `json:"capabilities,omitempty"`
	NoNewPrivileges *bool           `json:"noNewPrivileges,omitempty"`
	OOMScoreAdj     *int            `json:"oomScoreAdj,omitempty"`
	Scheduler       Scheduler       `json:"scheduler,omitempty"`
	SELinuxLabel    string          `json:"selinuxLabel,omitempty"`
	IOPriority      IOPriority      `json:"ioPriority,omitempty"`
	ExecCPUAffinity ExecCPUAffinity `json:"execCPUAffinity,omitempty"`
}

type RLimit struct {
	Type string `json:"type"`
	Soft uint64 `json:"soft"`
	Hard uint64 `json:"hard"`
}

type Capability struct {
	Effective   string `json:"effective"`
	Bounding    string `json:"bounding"`
	Inheritable string `json:"inheritable"`
	Permitted   string `json:"permitted"`
	Ambient     string `json:"ambient"`
}

type Scheduler struct {
	Policy   string   `json:"policy"`
	Nice     *int32   `json:"nice,omitempty"`
	Priority *int32   `json:"priority,omitempty"`
	Flags    []string `json:"flags,omitempty"`
	Runtime  *uint64  `json:"runtime,omitempty"`
	Deadline *uint64  `json:"deadline,omitempty"`
	Period   *uint64  `json:"period,omitempty"`
}

type IOPriority struct {
	Class    string `json:"class"`
	Priority int    `json:"priority"`
}

type ExecCPUAffinity struct {
	Initial string `json:"initial,omitempty"`
	Final   string `json:"final,omitempty"`
}

type User struct {
	UID            int   `json:"uid"`
	GID            int   `json:"gid"`
	Umask          *int  `json:"umask,omitempty"`
	AdditionalGids []int `json:"additionalGids,omitempty"`
}

type UIDMapping struct {
	ContainerID uint32 `json:"containerID"`
	HostID      uint32 `json:"hostID"`
	Size        uint32 `json:"size"`
}
