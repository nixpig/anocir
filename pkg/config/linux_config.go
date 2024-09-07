package config

type LinuxConfig struct {
	Namespaces  []Namespace  `json:"namespaces"`
	UIDMappings []UIDMapping `json:"uidMappings,omitempty"`
	GIDMappings []UIDMapping `json:"gidMappings,omitempty"`
	TimeOffsets struct {
		secs     int64
		nanosecs uint32
	} `json:"timeOffsets,omitempty"`
	Devices           []Device          `json:"devices,omitempty"`
	CGroupsPath       string            `json:"cgroupsPath,omitempty"`
	Resources         []LinuxResource   `json:"resources,omitempty"`
	Unified           map[string]string `json:"unified,omitempty"`
	IntelRDT          IntelRDT          `json:"intelRdt,omitempty"`
	SysCTL            map[string]string `json:"sysctl,omitempty"`
	SecComp           SecComp           `json:"seccomp,omitempty"`
	RootfsPropagation RootfsPropagation `json:"rootfsPropagation,omitempty"`
	MaskedPaths       []string          `json:"maskedPaths,omitempty"`
	ReadonlyPaths     []string          `json:"readonlyPaths,omitempty"`
	MountLabel        string            `json:"mountLabel,omitempty"`
	Personality       Personality       `json:"personality,omitempty"`
}

type Personality struct {
	Domain ProcessExecutionDomain `json:"domain"`
	Flags  []string               `json:"flags,omitempty"`
}

type Namespace struct {
	Type NamespaceType `json:"type"`
	Path string        `json:"path,omitempty"`
}

type Device struct {
	Type     DeviceType `json:"type"`
	Path     string     `json:"path"`
	Major    int64      `json:"major"`
	Minor    int64      `json:"minor"`
	FileMode *uint32    `json:"fileMode,omitempty"`
	Uid      *uint32    `json:"uid,omitempty"`
	Gid      *uint32    `json:"gid,omitempty"`
}

type LinuxResource struct {
	Memory         MemoryResource          `json:"memory,omitempty"`
	CPU            CPUResource             `json:"cpu,omitempty"`
	BlockIO        BlkioResource           `json:"blockIO,omitempty"`
	HugePageLimits []HugePageLimitResource `json:"hugePageLimits,omitempty"`
	Network        NetworkResource         `json:"network,omitempty"`
	Devices        []DeviceResource        `json:"devices,omitempty"`
	PIDs           PIDResource             `json:"pids,omitempty"`
	RDMA           map[string]RDMAResource `json:"rdma,omitempty"`
}

type IntelRDT struct {
	ClosID        string `json:"closID"`
	L3CacheSchema string `json:"l3CacheSchema"`
	MemBWSchema   string `json:"memBwSchema"`
}

type RDMAResource struct {
	HCAHandles *uint32 `json:"hcaHandles,omitempty"`
	HCAObjects *uint32 `json:"hcaObjects,omitempty"`
}

type PIDResource struct {
	Limit int64 `json:"limit"`
}

type NetworkResource struct {
	ClassID    *uint32           `json:"classID,omitempty"`
	Priorities []NetworkPriority `json:"priorities,omitempty"`
}

type HugePageLimitResource struct {
	PageSize string `json:"pageSize"`
	Limit    uint64 `json:"limit"`
}

type MemoryResource struct {
	Limit             *int64  `json:"limit,omitempty"`
	Reservation       *int64  `json:"reservation,omitempty"`
	Swap              *int64  `json:"swap,omitempty"`
	Kernel            *int64  `json:"kernel,omitempty"`
	KernelTCP         *int64  `json:"kernelTCP,omitempty"`
	Swappiness        *uint64 `json:"swappiness,omitempty"`
	DisableOOMKiller  *bool   `json:"disableOOMKiller,omitempty"`
	UseHierarchy      *bool   `json:"useHierarchy,omitempty"`
	CheckBeforeUpdate *bool   `json:"checkBeforeUpdate,omitempty"`
}

type DeviceResource struct {
	Allow  bool       `json:"allow"`
	Type   DeviceType `json:"type,omitempty"`
	Major  *int64     `json:"major,omitempty"`
	Minor  *int64     `json:"minor,omitempty"`
	Access string     `json:"access,omitempty"`
}

type BlkioResource struct {
	Weight                  *uint16         `json:"weight,omitempty"`
	LeafWeight              *uint16         `json:"leafWeight,omitempty"`
	WeightDevice            BandwidthWeight `json:"weightDevice,omitempty"`
	ThrottleReadBPSDevice   ThrottleRate    `json:"throttleReadBpsDevice,omitempty"`
	ThrottleWriteBPSDevice  ThrottleRate    `json:"throttleWriteBpsDevice,omitempty"`
	ThrottleReadIOPSDevice  ThrottleRate    `json:"throttleReadIOPSDevice,omitempty"`
	ThrottleWriteIOPSDevice ThrottleRate    `json:"throttleWriteIOPSDevice,omitempty"`
}

type CPUResource struct {
	Shares          *uint64 `json:"shares,omitempty"`
	Quota           *int64  `json:"quota,omitempty"`
	Burst           *uint64 `json:"burst,omitempty"`
	Period          *uint64 `json:"period,omitempty"`
	RealtimeRuntime *int64  `json:"realtimeRuntime,omitempty"`
	RealtimePeriod  *uint64 `json:"realtimePeriod,omitempty"`
	CPUs            string  `json:"cpus,omitempty"`
	Mems            string  `json:"mems,omitempty"`
	Idle            *int64  `json:"idle,omitempty"`
}

type BandwidthWeight struct {
	Major      int64   `json:"major"`
	Minor      int64   `json:"minor"`
	Weight     *uint16 `json:"weight,omitempty"`
	LeafWeight *uint16 `json:"leafWeight,omitempty"`
}

type ThrottleRate struct {
	Major *int64  `json:"major,omitempty"`
	Minor *int64  `json:"minor,omitempty"`
	Rate  *uint64 `json:"rate,omitempty"`
}

type NetworkPriority struct {
	Name     string `json:"name"`
	Priority uint32 `json:"priority"`
}

type SecCompArchitecture string

const (
	SCMP_ARCH_X86         SecCompArchitecture = "SCMP_ARCH_X86"
	SCMP_ARCH_X86_64      SecCompArchitecture = "SCMP_ARCH_X86_64"
	SCMP_ARCH_X32         SecCompArchitecture = "SCMP_ARCH_X32"
	SCMP_ARCH_ARM         SecCompArchitecture = "SCMP_ARCH_ARM"
	SCMP_ARCH_AARCH64     SecCompArchitecture = "SCMP_ARCH_AARCH64"
	SCMP_ARCH_MIPS        SecCompArchitecture = "SCMP_ARCH_MIPS"
	SCMP_ARCH_MIPS64      SecCompArchitecture = "SCMP_ARCH_MIPS64"
	SCMP_ARCH_MIPS64N32   SecCompArchitecture = "SCMP_ARCH_MIPS64N32"
	SCMP_ARCH_MIPSEL      SecCompArchitecture = "SCMP_ARCH_MIPSEL"
	SCMP_ARCH_MIPSEL64    SecCompArchitecture = "SCMP_ARCH_MIPSEL64"
	SCMP_ARCH_MIPSEL64N32 SecCompArchitecture = "SCMP_ARCH_MIPSEL64N32"
	SCMP_ARCH_PPC         SecCompArchitecture = "SCMP_ARCH_PPC"
	SCMP_ARCH_PPC64       SecCompArchitecture = "SCMP_ARCH_PPC64"
	SCMP_ARCH_PPC64LE     SecCompArchitecture = "SCMP_ARCH_PPC64LE"
	SCMP_ARCH_S390        SecCompArchitecture = "SCMP_ARCH_S390"
	SCMP_ARCH_S390X       SecCompArchitecture = "SCMP_ARCH_S390X"
	SCMP_ARCH_PARISC      SecCompArchitecture = "SCMP_ARCH_PARISC"
	SCMP_ARCH_PARISC64    SecCompArchitecture = "SCMP_ARCH_PARISC64"
	SCMP_ARCH_RISCV64     SecCompArchitecture = "SCMP_ARCH_RISCV64"
)

type SecCompFlag string

const (
	SECCOMP_FILTER_FLAG_TSYNC              SecCompFlag = "SECCOMP_FILTER_FLAG_TSYNC"
	SECCOMP_FILTER_FLAG_LOG                SecCompFlag = "SECCOMP_FILTER_FLAG_LOG"
	SECCOMP_FILTER_FLAG_SPEC_ALLOW         SecCompFlag = "SECCOMP_FILTER_FLAG_SPEC_ALLOW"
	SECCOMP_FILTER_FLAG_WAIT_KILLABLE_RECV SecCompFlag = "SECCOMP_FILTER_FLAG_WAIT_KILLABLE_RECV"
)

type SecCompSyscallAction string

const (
	SCMP_ACT_KILL         SecCompSyscallAction = "SCMP_ACT_KILL"
	SCMP_ACT_KILL_PROCESS SecCompSyscallAction = "SCMP_ACT_KILL_PROCESS"
	SCMP_ACT_KILL_THREAD  SecCompSyscallAction = "SCMP_ACT_KILL_THREAD"
	SCMP_ACT_TRAP         SecCompSyscallAction = "SCMP_ACT_TRAP"
	SCMP_ACT_ERRNO        SecCompSyscallAction = "SCMP_ACT_ERRNO"
	SCMP_ACT_TRACE        SecCompSyscallAction = "SCMP_ACT_TRACE"
	SCMP_ACT_ALLOW        SecCompSyscallAction = "SCMP_ACT_ALLOW"
	SCMP_ACT_LOG          SecCompSyscallAction = "SCMP_ACT_LOG"
	SCMP_ACT_NOTIFY       SecCompSyscallAction = "SCMP_ACT_NOTIFY"
)

type SecCompSyscallOp string

const (
	SCMP_CMP_NE        SecCompSyscallOp = "SCMP_CMP_NE"
	SCMP_CMP_LT        SecCompSyscallOp = "SCMP_CMP_LT"
	SCMP_CMP_LE        SecCompSyscallOp = "SCMP_CMP_LE"
	SCMP_CMP_EQ        SecCompSyscallOp = "SCMP_CMP_EQ"
	SCMP_CMP_GE        SecCompSyscallOp = "SCMP_CMP_GE"
	SCMP_CMP_GT        SecCompSyscallOp = "SCMP_CMP_GT"
	SCMP_CMP_MASKED_EQ SecCompSyscallOp = "SCMP_CMP_MASKED_EQ"
)

type SecCompSyscallArg struct {
	Index    uint             `json:"index"`
	Value    uint64           `json:"value"`
	ValueTwo uint64           `json:"valueTwo"`
	Op       SecCompSyscallOp `json:"op"`
}

type SecCompSyscall struct {
	Names    []string             `json:"names"`
	Action   SecCompSyscallAction `json:"action"`
	ErrnoRet uint                 `json:"errnoRet"`
	Args     []SecCompSyscallArg  `json:"args"`
}

type SecComp struct {
	DefaultAction    SecCompSyscallAction  `json:"defaultAction"`
	DefaultErrnoRet  *uint                 `json:"defaultErrnoRet,omitempty"`
	Architectures    []SecCompArchitecture `json:"architectures,omitempty"`
	Flags            []SecCompFlag         `json:"flags,omitempty"`
	ListenerPath     string                `json:"listenerPath,omitempty"`
	ListenerMetaData string                `json:"listenerMetaData,omitempty"`
	Syscalls         []SecCompSyscall      `json:"syscalls,omitempty"`
}

type RootfsPropagation string

const (
	Shared     RootfsPropagation = "shared"
	Slave      RootfsPropagation = "slave"
	Private    RootfsPropagation = "private"
	Unbindable RootfsPropagation = "unbindable"
)

type ProcessExecutionDomain string

const (
	Linux   ProcessExecutionDomain = "LINUX"
	Linux32 ProcessExecutionDomain = "LINUX32"
)
