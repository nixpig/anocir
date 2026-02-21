package platform

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

// Cap represents a Linux capability.
type Cap int

const (
	CAP_CHOWN              Cap = 0
	CAP_DAC_OVERRIDE       Cap = 1
	CAP_DAC_READ_SEARCH    Cap = 2
	CAP_FOWNER             Cap = 3
	CAP_FSETID             Cap = 4
	CAP_KILL               Cap = 5
	CAP_SETGID             Cap = 6
	CAP_SETUID             Cap = 7
	CAP_SETPCAP            Cap = 8
	CAP_LINUX_IMMUTABLE    Cap = 9
	CAP_NET_BIND_SERVICE   Cap = 10
	CAP_NET_BROADCAST      Cap = 11
	CAP_NET_ADMIN          Cap = 12
	CAP_NET_RAW            Cap = 13
	CAP_IPC_LOCK           Cap = 14
	CAP_IPC_OWNER          Cap = 15
	CAP_SYS_MODULE         Cap = 16
	CAP_SYS_RAWIO          Cap = 17
	CAP_SYS_CHROOT         Cap = 18
	CAP_SYS_PTRACE         Cap = 19
	CAP_SYS_PACCT          Cap = 20
	CAP_SYS_ADMIN          Cap = 21
	CAP_SYS_BOOT           Cap = 22
	CAP_SYS_NICE           Cap = 23
	CAP_SYS_RESOURCE       Cap = 24
	CAP_SYS_TIME           Cap = 25
	CAP_SYS_TTY_CONFIG     Cap = 26
	CAP_MKNOD              Cap = 27
	CAP_LEASE              Cap = 28
	CAP_AUDIT_WRITE        Cap = 29
	CAP_AUDIT_CONTROL      Cap = 30
	CAP_SETFCAP            Cap = 31
	CAP_MAC_OVERRIDE       Cap = 32
	CAP_MAC_ADMIN          Cap = 33
	CAP_SYSLOG             Cap = 34
	CAP_WAKE_ALARM         Cap = 35
	CAP_BLOCK_SUSPEND      Cap = 36
	CAP_AUDIT_READ         Cap = 37
	CAP_PERFMON            Cap = 38
	CAP_BPF                Cap = 39
	CAP_CHECKPOINT_RESTORE Cap = 40
	CAP_LAST_CAP           Cap = 63
)

// capabilities maps CAP_* capability name strings to their corresponding
// Cap constant values.
var capabilities = map[string]Cap{
	"CAP_AUDIT_CONTROL":      CAP_AUDIT_CONTROL,
	"CAP_AUDIT_READ":         CAP_AUDIT_READ,
	"CAP_AUDIT_WRITE":        CAP_AUDIT_WRITE,
	"CAP_BLOCK_SUSPEND":      CAP_BLOCK_SUSPEND,
	"CAP_BPF":                CAP_BPF,
	"CAP_CHECKPOINT_RESTORE": CAP_CHECKPOINT_RESTORE,
	"CAP_CHOWN":              CAP_CHOWN,
	"CAP_DAC_OVERRIDE":       CAP_DAC_OVERRIDE,
	"CAP_DAC_READ_SEARCH":    CAP_DAC_READ_SEARCH,
	"CAP_FOWNER":             CAP_FOWNER,
	"CAP_FSETID":             CAP_FSETID,
	"CAP_IPC_LOCK":           CAP_IPC_LOCK,
	"CAP_IPC_OWNER":          CAP_IPC_OWNER,
	"CAP_KILL":               CAP_KILL,
	"CAP_LEASE":              CAP_LEASE,
	"CAP_LINUX_IMMUTABLE":    CAP_LINUX_IMMUTABLE,
	"CAP_MAC_ADMIN":          CAP_MAC_ADMIN,
	"CAP_MAC_OVERRIDE":       CAP_MAC_OVERRIDE,
	"CAP_MKNOD":              CAP_MKNOD,
	"CAP_NET_ADMIN":          CAP_NET_ADMIN,
	"CAP_NET_BIND_SERVICE":   CAP_NET_BIND_SERVICE,
	"CAP_NET_BROADCAST":      CAP_NET_BROADCAST,
	"CAP_NET_RAW":            CAP_NET_RAW,
	"CAP_PERFMON":            CAP_PERFMON,
	"CAP_SETGID":             CAP_SETGID,
	"CAP_SETFCAP":            CAP_SETFCAP,
	"CAP_SETPCAP":            CAP_SETPCAP,
	"CAP_SETUID":             CAP_SETUID,
	"CAP_SYS_ADMIN":          CAP_SYS_ADMIN,
	"CAP_SYS_BOOT":           CAP_SYS_BOOT,
	"CAP_SYS_CHROOT":         CAP_SYS_CHROOT,
	"CAP_SYS_MODULE":         CAP_SYS_MODULE,
	"CAP_SYS_NICE":           CAP_SYS_NICE,
	"CAP_SYS_PACCT":          CAP_SYS_PACCT,
	"CAP_SYS_PTRACE":         CAP_SYS_PTRACE,
	"CAP_SYS_RAWIO":          CAP_SYS_RAWIO,
	"CAP_SYS_RESOURCE":       CAP_SYS_RESOURCE,
	"CAP_SYS_TIME":           CAP_SYS_TIME,
	"CAP_SYS_TTY_CONFIG":     CAP_SYS_TTY_CONFIG,
	"CAP_SYSLOG":             CAP_SYSLOG,
	"CAP_WAKE_ALARM":         CAP_WAKE_ALARM,
}

// SetAllCapabilities sets all available capabilities.
func SetAllCapabilities() error {
	allCaps := slices.Collect(maps.Keys(capabilities))

	return SetCapabilities(&specs.LinuxCapabilities{
		Bounding:    allCaps,
		Effective:   allCaps,
		Inheritable: allCaps,
		Permitted:   allCaps,
		Ambient:     allCaps,
	})
}

// SetCapabilities sets the process' effective, inheritable, permitted and
// ambient capabilities based on the provided caps.
//
// Effective, permitted and inheritable capabilities are set using the capset syscall.
// Ambient capabilities are set using the prctl syscall by first clearing all
// ambients capabilities and then raising the ones we want.
func SetCapabilities(caps *specs.LinuxCapabilities) error {
	var effective, permitted, inheritable [2]uint32

	if caps.Effective != nil {
		for _, c := range resolveCaps(caps.Effective) {
			capShift(&effective, uint32(c))
		}
	}

	if caps.Permitted != nil {
		for _, c := range resolveCaps(caps.Permitted) {
			capShift(&permitted, uint32(c))
		}
	}

	if caps.Inheritable != nil {
		for _, c := range resolveCaps(caps.Inheritable) {
			capShift(&inheritable, uint32(c))
		}
	}

	header := unix.CapUserHeader{
		Version: unix.LINUX_CAPABILITY_VERSION_3,
		Pid:     0, // 0 means current thread.
	}

	data := [2]unix.CapUserData{
		{Effective: effective[0], Permitted: permitted[0], Inheritable: inheritable[0]},
		{Effective: effective[1], Permitted: permitted[1], Inheritable: inheritable[1]},
	}

	if err := unix.Capset(&header, &data[0]); err != nil {
		return fmt.Errorf("capset: %w", err)
	}

	if err := unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0); err != nil {
		// Ignore unsupported ambient caps.
		if err != unix.EINVAL {
			return fmt.Errorf("clear ambient caps: %w", err)
		}
	}

	if caps.Ambient != nil {
		for _, c := range resolveCaps(caps.Ambient) {
			if err := unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_RAISE, uintptr(c), 0, 0); err != nil {
				// Ignore if this specific cap can't be raised.
				if err != unix.EPERM && err != unix.EINVAL {
					return fmt.Errorf("raise ambient cap %d: %w", c, err)
				}
			}
		}
	}

	return nil
}

// DropBoundingCapabilities drops the bounding capabilities specified in caps
// from the current thread's capability bounding set.
func DropBoundingCapabilities(caps *specs.LinuxCapabilities) error {
	if caps.Bounding != nil {
		retain := make(map[Cap]struct{})

		for _, c := range resolveCaps(caps.Bounding) {
			retain[c] = struct{}{}
		}

		for c := Cap(0); c <= CAP_LAST_CAP; c++ {
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

// SetKeepCaps sets the state of the 'keep capabilities' flag using
// PR_SET_KEEPCAPS. 0 = clear flag, 1 = set flag.
func SetKeepCaps(state uintptr) error {
	if err := unix.Prctl(unix.PR_SET_KEEPCAPS, state, 0, 0, 0); err != nil {
		return fmt.Errorf("set keep capabilities flag: %w", err)
	}

	return nil
}

// resolveCaps converts a slice of capability name strings to a slice of their
// corresponding Cap values. If a capability name can't be mapped, a warning is
// logged and the capability is skipped.
func resolveCaps(names []string) []Cap {
	resolved := []Cap{}

	for _, name := range names {
		if name == "ALL" || name == "CAP_ALL" {
			resolved = slices.Collect(maps.Values(capabilities))
			break
		}

		if v, ok := capabilities[name]; ok {
			resolved = append(resolved, v)
		} else {
			// Spec requires printing "Warning: ..." to stderr.
			fmt.Fprintf(os.Stderr, "Warning: capability %s cannot be mapped\n", name)
			slog.Warn("capability cannot be mapped", "name", name)
		}
	}

	return resolved
}

// capShift performs the bitshifting to build the capability bitmasks.
// Two 32-bit words for 64-bit caps.
func capShift(caparr *[2]uint32, resolved uint32) {
	if resolved < 32 {
		caparr[0] |= 1 << (resolved)
	} else {
		caparr[1] |= 1 << (resolved - 32)
	}
}
