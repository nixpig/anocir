package config

import "github.com/nixpig/brownie/pkg"

type NamespaceType string
type DeviceType string

const (
	PID     NamespaceType = "pid"
	Network NamespaceType = "network"
	mount   NamespaceType = "mount"
	IPC     NamespaceType = "ipc"
	UTS     NamespaceType = "uts"
	user    NamespaceType = "user"
	CGroup  NamespaceType = "cgroup"
	Time    NamespaceType = "time"

	AllDevices           DeviceType = "a"
	BlockDevice          DeviceType = "b"
	CharDevice           DeviceType = "c"
	UnbufferedCharDevice DeviceType = "u"
	FifoDevice           DeviceType = "p"
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
	Capabilities    Capabilities    `json:"capabilities,omitempty"`
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

type Capabilities struct {
	Effective   []string `json:"effective,omitempty"`
	Bounding    []string `json:"bounding,omitempty"`
	Inheritable []string `json:"inheritable,omitempty"`
	Permitted   []string `json:"permitted,omitempty"`
	Ambient     []string `json:"ambient,omitempty"`
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
