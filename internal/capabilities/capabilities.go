package capabilities

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/syndtr/gocapability/capability"
)

var Capabilities = map[string]capability.Cap{
	"CAP_AUDIT_CONTROL":      capability.CAP_AUDIT_CONTROL,
	"CAP_AUDIT_READ":         capability.CAP_AUDIT_READ,
	"CAP_AUDIT_WRITE":        capability.CAP_AUDIT_WRITE,
	"CAP_BLOCK_SUSPEND":      capability.CAP_BLOCK_SUSPEND,
	"CAP_BPF":                capability.CAP_BPF,
	"CAP_CHECKPOINT_RESTORE": capability.CAP_CHECKPOINT_RESTORE,
	"CAP_CHOWN":              capability.CAP_CHOWN,
	"CAP_DAC_OVERRIDE":       capability.CAP_DAC_OVERRIDE,
	"CAP_DAC_READ_SEARCH":    capability.CAP_DAC_READ_SEARCH,
	"CAP_FOWNER":             capability.CAP_FOWNER,
	"CAP_FSETID":             capability.CAP_FSETID,
	"CAP_IPC_LOCK":           capability.CAP_IPC_LOCK,
	"CAP_IPC_OWNER":          capability.CAP_IPC_OWNER,
	"CAP_KILL":               capability.CAP_KILL,
	"CAP_LEASE":              capability.CAP_LEASE,
	"CAP_LINUX_IMMUTABLE":    capability.CAP_LINUX_IMMUTABLE,
	"CAP_MAC_ADMIN":          capability.CAP_MAC_ADMIN,
	"CAP_MAC_OVERRIDE":       capability.CAP_MAC_OVERRIDE,
	"CAP_MKNOD":              capability.CAP_MKNOD,
	"CAP_NET_ADMIN":          capability.CAP_NET_ADMIN,
	"CAP_NET_BIND_SERVICE":   capability.CAP_NET_BIND_SERVICE,
	"CAP_NET_BROADCAST":      capability.CAP_NET_BROADCAST,
	"CAP_NET_RAW":            capability.CAP_NET_RAW,
	"CAP_PERFMON":            capability.CAP_PERFMON,
	"CAP_SETGID":             capability.CAP_SETGID,
	"CAP_SETFCAP":            capability.CAP_SETFCAP,
	"CAP_SETPCAP":            capability.CAP_SETPCAP,
	"CAP_SETUID":             capability.CAP_SETUID,
	"CAP_SYS_ADMIN":          capability.CAP_SYS_ADMIN,
	"CAP_SYS_BOOT":           capability.CAP_SYS_BOOT,
	"CAP_SYS_CHROOT":         capability.CAP_SYS_CHROOT,
	"CAP_SYS_MODULE":         capability.CAP_SYS_MODULE,
	"CAP_SYS_NICE":           capability.CAP_SYS_NICE,
	"CAP_SYS_PACCT":          capability.CAP_SYS_PACCT,
	"CAP_SYS_PTRACE":         capability.CAP_SYS_PTRACE,
	"CAP_SYS_RAWIO":          capability.CAP_SYS_RAWIO,
	"CAP_SYS_RESOURCE":       capability.CAP_SYS_RESOURCE,
	"CAP_SYS_TIME":           capability.CAP_SYS_TIME,
	"CAP_SYS_TTY_CONFIG":     capability.CAP_SYS_TTY_CONFIG,
	"CAP_SYSLOG":             capability.CAP_SYSLOG,
	"CAP_WAKE_ALARM":         capability.CAP_WAKE_ALARM,
}

func SetCapabilities(caps *specs.LinuxCapabilities) error {
	if caps == nil {
		return nil
	}

	c, err := capability.NewPid2(0)
	if err != nil {
		return fmt.Errorf("initialise capabilities object: %w", err)
	}

	c.Clear(capability.BOUNDING)
	c.Clear(capability.EFFECTIVE)
	c.Clear(capability.INHERITABLE)
	c.Clear(capability.PERMITTED)
	c.Clear(capability.AMBIENT)

	if caps.Ambient != nil {
		for _, e := range caps.Ambient {
			if v, ok := Capabilities[e]; ok {
				c.Set(capability.AMBIENT, capability.Cap(v))
			}
		}
	}

	if caps.Bounding != nil {
		for _, e := range caps.Bounding {
			if v, ok := Capabilities[e]; ok {
				c.Set(capability.BOUNDING, capability.Cap(v))
			}
		}
	}

	if caps.Effective != nil {
		for _, e := range caps.Effective {
			if v, ok := Capabilities[e]; ok {
				c.Set(capability.EFFECTIVE, capability.Cap(v))
			}
		}
	}

	if caps.Permitted != nil {
		for _, e := range caps.Permitted {
			if v, ok := Capabilities[e]; ok {
				c.Set(capability.PERMITTED, capability.Cap(v))
			}
		}
	}

	if caps.Inheritable != nil {
		for _, e := range caps.Inheritable {
			if v, ok := Capabilities[e]; ok {
				c.Set(capability.INHERITABLE, capability.Cap(v))
			}
		}
	}

	if err := c.Apply(
		capability.INHERITABLE |
			capability.EFFECTIVE |
			capability.BOUNDING |
			capability.PERMITTED |
			capability.AMBIENT,
	); err != nil {
		return fmt.Errorf("apply capabilities: %w", err)
	}

	return nil
}
