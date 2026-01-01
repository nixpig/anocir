package platform

import (
	"fmt"
	"os"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/syndtr/gocapability/capability"
	"golang.org/x/sys/unix"
)

// capabilities maps CAP_* capability name strings to their corresponding
// capability.Cap constant values.
var capabilities = map[string]capability.Cap{
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

// SetCapabilities sets the process capabilities based on the provided
// LinuxCapabilities.
func SetCapabilities(caps *specs.LinuxCapabilities) error {
	c, err := capability.NewPid2(0)
	if err != nil {
		return fmt.Errorf("initialise capabilities object: %w", err)
	}

	if err := c.Load(); err != nil {
		return fmt.Errorf("load capabilities: %w", err)
	}

	c.Clear(capability.EFFECTIVE)
	c.Clear(capability.INHERITABLE)
	c.Clear(capability.PERMITTED)
	c.Clear(capability.AMBIENT)

	if caps.Ambient != nil {
		c.Set(capability.AMBIENT, resolveCaps(caps.Ambient)...)
	}

	if caps.Effective != nil {
		c.Set(capability.EFFECTIVE, resolveCaps(caps.Effective)...)
	}

	if caps.Permitted != nil {
		c.Set(capability.PERMITTED, resolveCaps(caps.Permitted)...)
	}

	if caps.Inheritable != nil {
		c.Set(capability.INHERITABLE, resolveCaps(caps.Inheritable)...)
	}

	if err := c.Apply(
		capability.INHERITABLE |
			capability.EFFECTIVE |
			capability.PERMITTED |
			capability.AMBIENT,
	); err != nil {
		return fmt.Errorf("apply capabilities: %w", err)
	}

	return nil
}

func DropBoundingCapabilities(caps *specs.LinuxCapabilities) error {
	if caps.Bounding != nil {
		retain := make(map[capability.Cap]struct{})

		for _, c := range resolveCaps(caps.Bounding) {
			retain[c] = struct{}{}
		}

		for c := capability.Cap(0); c <= capability.CAP_LAST_CAP; c++ {
			if _, ok := retain[c]; ok {
				continue
			}

			if err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(c), 0, 0, 0); err != nil {
				if err != unix.EINVAL {
					return fmt.Errorf("drop bounding cap '%d': %w", c, err)
				}
			}
		}
	}

	return nil
}

// resolveCaps converts a slice of capability name strings to a slice of their
// corresponding capability.Cap values. If a capability name can't be mapped,
// a warning is logged and the capability is skipped.
func resolveCaps(names []string) []capability.Cap {
	resolved := []capability.Cap{}

	for _, name := range names {
		if v, ok := capabilities[name]; ok {
			resolved = append(resolved, v)
		} else {
			fmt.Fprintf(os.Stdout, "Warning: capability %s cannot be mapped\n", name)
		}
	}

	return resolved
}
