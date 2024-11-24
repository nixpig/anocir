package cgroups

import (
	"fmt"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

var Rlimits = map[string]uint{
	"RLIMIT_AS":     syscall.RLIMIT_AS,
	"RLIMIT_CORE":   syscall.RLIMIT_CORE,
	"RLIMIT_CPU":    syscall.RLIMIT_CPU,
	"RLIMIT_DATA":   syscall.RLIMIT_DATA,
	"RLIMIT_FSIZE":  syscall.RLIMIT_FSIZE,
	"RLIMIT_STACK":  syscall.RLIMIT_STACK,
	"RLIMIT_NOFILE": syscall.RLIMIT_NOFILE,
}

func SetRlimits(rlimits []specs.POSIXRlimit) error {
	for _, rl := range rlimits {
		rlType := int(Rlimits[rl.Type])
		if err := syscall.Getrlimit(rlType, &syscall.Rlimit{
			Cur: rl.Soft,
			Max: rl.Hard,
		}); err != nil {
			return fmt.Errorf("map rlimit to kernel interface: %w", err)
		}

		if err := syscall.Getrlimit(rlType, &syscall.Rlimit{
			Cur: rl.Soft,
			Max: rl.Hard,
		}); err != nil {
			return fmt.Errorf("map rlimit to kernel interface: %w", err)
		}

		if err := syscall.Setrlimit(rlType, &syscall.Rlimit{
			Cur: rl.Soft,
			Max: rl.Hard,
		}); err != nil {
			return fmt.Errorf("set rlimit: %w", err)
		}
	}

	return nil
}
