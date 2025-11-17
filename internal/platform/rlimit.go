package platform

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var rlimit = map[string]uint{
	"RLIMIT_AS":         unix.RLIMIT_AS,
	"RLIMIT_CORE":       unix.RLIMIT_CORE,
	"RLIMIT_CPU":        unix.RLIMIT_CPU,
	"RLIMIT_DATA":       unix.RLIMIT_DATA,
	"RLIMIT_FSIZE":      unix.RLIMIT_FSIZE,
	"RLIMIT_STACK":      unix.RLIMIT_STACK,
	"RLIMIT_NOFILE":     unix.RLIMIT_NOFILE,
	"RLIMIT_RSS":        unix.RLIMIT_RSS,
	"RLIMIT_NPROC":      unix.RLIMIT_NPROC,
	"RLIMIT_MEMLOCK":    unix.RLIMIT_MEMLOCK,
	"RLIMIT_LOCKS":      unix.RLIMIT_LOCKS,
	"RLIMIT_SIGPENDING": unix.RLIMIT_SIGPENDING,
	"RLIMIT_MSGQUEUE":   unix.RLIMIT_MSGQUEUE,
	"RLIMIT_NICE":       unix.RLIMIT_NICE,
	"RLIMIT_RTPRIO":     unix.RLIMIT_RTPRIO,
	"RLIMIT_RTTIME":     unix.RLIMIT_RTTIME,
}

// SetRlimits sets the resource limits for the current (container) process
// based on the given POSIXRlimit specifications.
func SetRlimits(rlimits []specs.POSIXRlimit) error {
	for _, rl := range rlimits {
		if err := unix.Setrlimit(int(rlimit[rl.Type]), &unix.Rlimit{
			Cur: rl.Soft,
			Max: rl.Hard,
		}); err != nil {
			return fmt.Errorf("set rlimit: %w", err)
		}
	}

	return nil
}
