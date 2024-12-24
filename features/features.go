package features

func Get() *Features {
	return &Features{
		OCIVersionMin: "1.0.1-dev",
		OCIVersionMax: "1.0.1-dev",
		Hooks: []string{
			"prestart",
			"createRuntime",
			"createContainer",
			"startContainer",
			"poststart",
			"poststop",
		},
		MountOptions: []string{
			"async",
			"atime",
			"bind",
			"defaults",
			"dev",
			"diratime",
			"dirsync",
			"exec",
			"iversion",
			"lazytime",
			"loud",
			"noatime",
			"nodev",
			"nodiratime",
			"noexec",
			"noiversion",
			"nolazytime",
			"norelatime",
			"nostrictatime",
			"nosuid",
			"nosymfollow",
			"private",
			"rbind",
			"relatime",
			"remount",
			"ro",
			"rprivate",
			"rshared",
			"rslave",
			"runbindable",
			"rw",
			"shared",
			"silent",
			"slave",
			"strictatime",
			"suid",
			"sync",
			"unbindable",
		},
		Linux: &LinuxFeatures{
			Namespaces: []string{
				"cgroup",
				"ipc",
				"mount",
				"network",
				"pid",
				"user",
				"uts",
			},
			Capabilities: []string{
				"CAP_AUDIT_CONTROL",
				"CAP_AUDIT_READ",
				"CAP_AUDIT_WRITE",
				"CAP_BLOCK_SUSPEND",
				"CAP_BPF",
				"CAP_CHECKPOINT_RESTORE",
				"CAP_CHOWN",
				"CAP_DAC_OVERRIDE",
				"CAP_DAC_READ_SEARCH",
				"CAP_FOWNER",
				"CAP_FSETID",
				"CAP_IPC_LOCK",
				"CAP_IPC_OWNER",
				"CAP_KILL",
				"CAP_LEASE",
				"CAP_LINUX_IMMUTABLE",
				"CAP_MAC_ADMIN",
				"CAP_MAC_OVERRIDE",
				"CAP_MKNOD",
				"CAP_NET_ADMIN",
				"CAP_NET_BIND_SERVICE",
				"CAP_NET_BROADCAST",
				"CAP_NET_RAW",
				"CAP_PERFMON",
				"CAP_SETGID",
				"CAP_SETFCAP",
				"CAP_SETPCAP",
				"CAP_SETUID",
				"CAP_SYS_ADMIN",
				"CAP_SYS_BOOT",
				"CAP_SYS_CHROOT",
				"CAP_SYS_MODULE",
				"CAP_SYS_NICE",
				"CAP_SYS_PACCT",
				"CAP_SYS_PTRACE",
				"CAP_SYS_RAWIO",
				"CAP_SYS_RESOURCE",
				"CAP_SYS_TIME",
				"CAP_SYS_TTY_CONFIG",
				"CAP_SYSLOG",
				"CAP_WAKE_ALARM",
			},
			CGroup: &CGroupFeatures{
				V1:          true,
				V2:          false,
				Systemd:     true,
				SystemdUser: false,
				RDMA:        false,
			},
			Seccomp: &SeccompFeatures{
				Enabled: false,
			},
			AppArmor: &AppArmorFeatures{
				Enabled: false,
			},
			SELinux: &SELinuxFeatures{
				Enabled: false,
			},
			IntelRDT: &IntelRDTFeatures{
				Enabled: false,
			},
		},
	}
}

type Features struct {
	OCIVersionMin string            `json:"ociVersionMin"`
	OCIVersionMax string            `json:"ociVersionMax"`
	Hooks         []string          `json:"hooks,omitempty"`
	MountOptions  []string          `json:"mountOptions,omitempty"`
	Linux         *LinuxFeatures    `json:"linux,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

type LinuxFeatures struct {
	Namespaces      []string                 `json:"namespaces,omitempty"`
	Capabilities    []string                 `json:"capabilities,omitempty"`
	CGroup          *CGroupFeatures          `json:"cgroup,omitempty"`
	Seccomp         *SeccompFeatures         `json:"seccomp,omitempty"`
	AppArmor        *AppArmorFeatures        `json:"apparmor,omitempty"`
	SELinux         *SELinuxFeatures         `json:"selinux,omitempty"`
	IntelRDT        *IntelRDTFeatures        `json:"intelRdt,omitempty"`
	MountEntensions *MountExtensionsFeatures `json:"mountExtensions,omitempty"`
}

type CGroupFeatures struct {
	V1          bool `json:"v1"`
	V2          bool `json:"v2"`
	Systemd     bool `json:"systemd"`
	SystemdUser bool `json:"systemdUser"`
	RDMA        bool `json:"rdma"`
}

type SeccompFeatures struct {
	Enabled        bool     `json:"enabled"`
	Actions        []string `json:"actions,omitempty"`
	Operators      []string `json:"operators,omitempty"`
	Archs          []string `json:"archs,omitempty"`
	KnownFlags     []string `json:"knownFlags,omitempty"`
	SupportedFlags []string `json:"supportedFlags,omitempty"`
}

type AppArmorFeatures struct {
	Enabled bool `json:"enabled"`
}

type SELinuxFeatures struct {
	Enabled bool `json:"enabled"`
}

type IntelRDTFeatures struct {
	Enabled bool `json:"enabled"`
}

type MountExtensionsFeatures struct {
	IDMap *IDMapFeatures `json:"idmap,omitempty"`
}

type IDMapFeatures struct {
	Enabled bool `json:"enabled"`
}
