package platform

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/opencontainers/runtime-spec/specs-go"
	libseccomp "github.com/seccomp/libseccomp-golang"
	"golang.org/x/sys/unix"
)

var seccompActions = map[specs.LinuxSeccompAction]libseccomp.ScmpAction{
	specs.ActKill:        libseccomp.ActKillThread,
	specs.ActKillProcess: libseccomp.ActKillProcess,
	specs.ActTrap:        libseccomp.ActTrap,
	specs.ActErrno:       libseccomp.ActErrno,
	specs.ActTrace:       libseccomp.ActTrace,
	specs.ActAllow:       libseccomp.ActAllow,
	specs.ActLog:         libseccomp.ActLog,
	specs.ActNotify:      libseccomp.ActNotify,
}

var seccompOperators = map[specs.LinuxSeccompOperator]libseccomp.ScmpCompareOp{
	specs.OpNotEqual:     libseccomp.CompareNotEqual,
	specs.OpLessThan:     libseccomp.CompareLess,
	specs.OpLessEqual:    libseccomp.CompareLessOrEqual,
	specs.OpEqualTo:      libseccomp.CompareEqual,
	specs.OpGreaterEqual: libseccomp.CompareGreaterEqual,
	specs.OpGreaterThan:  libseccomp.CompareGreater,
	specs.OpMaskedEqual:  libseccomp.CompareMaskedEqual,
}

var seccompArch = map[specs.Arch]libseccomp.ScmpArch{
	specs.ArchX86:         libseccomp.ArchX86,
	specs.ArchX86_64:      libseccomp.ArchAMD64,
	specs.ArchX32:         libseccomp.ArchX32,
	specs.ArchARM:         libseccomp.ArchARM,
	specs.ArchAARCH64:     libseccomp.ArchARM64,
	specs.ArchMIPS:        libseccomp.ArchMIPS,
	specs.ArchMIPS64:      libseccomp.ArchMIPS64,
	specs.ArchMIPS64N32:   libseccomp.ArchMIPS64N32,
	specs.ArchMIPSEL:      libseccomp.ArchMIPSEL,
	specs.ArchMIPSEL64:    libseccomp.ArchMIPSEL64,
	specs.ArchMIPSEL64N32: libseccomp.ArchMIPSEL64N32,
	specs.ArchPPC:         libseccomp.ArchPPC,
	specs.ArchPPC64:       libseccomp.ArchPPC64,
	specs.ArchPPC64LE:     libseccomp.ArchPPC64LE,
	specs.ArchS390:        libseccomp.ArchS390,
	specs.ArchS390X:       libseccomp.ArchS390X,
	specs.ArchRISCV64:     libseccomp.ArchRISCV64,
}

func mapSeccompAction(
	action specs.LinuxSeccompAction,
) libseccomp.ScmpAction {
	act, ok := seccompActions[action]
	if !ok {
		return libseccomp.ActInvalid
	}

	return act
}

func mapSeccompOperator(
	operator specs.LinuxSeccompOperator,
) libseccomp.ScmpCompareOp {
	op, ok := seccompOperators[operator]
	if !ok {
		return libseccomp.CompareInvalid
	}

	return op
}

func mapSeccompArch(arch specs.Arch) libseccomp.ScmpArch {
	a, ok := seccompArch[arch]
	if !ok {
		return libseccomp.ArchInvalid
	}

	return a
}

func buildSeccompFilter(
	spec *specs.LinuxSeccomp,
) (*libseccomp.ScmpFilter, error) {
	slog.Debug("build seccomp filter", "default_action", spec.DefaultAction)
	defaultAction := mapSeccompAction(spec.DefaultAction)

	if spec.DefaultAction == specs.ActErrno {
		errno := int16(unix.EPERM)
		if spec.DefaultErrnoRet != nil {
			errno = int16(*spec.DefaultErrnoRet)
		}
		defaultAction = defaultAction.SetReturnCode(errno)
	}

	filter, err := libseccomp.NewFilter(defaultAction)
	if err != nil {
		return nil, fmt.Errorf("new seccomp filter: %w", err)
	}

	for _, arch := range spec.Architectures {
		a := mapSeccompArch(arch)
		if err := filter.AddArch(a); err != nil {
			filter.Release()
			return nil, fmt.Errorf("add seccomp arch: %w", err)
		}
	}

	for _, sc := range spec.Syscalls {
		action := mapSeccompAction(sc.Action)

		if sc.Action == specs.ActErrno {
			errno := int16(unix.EPERM)
			if sc.ErrnoRet != nil {
				errno = int16(*sc.ErrnoRet)
			} else if spec.DefaultErrnoRet != nil {
				errno = int16(*spec.DefaultErrnoRet)
			}

			action = action.SetReturnCode(errno)
		}

		for _, name := range sc.Names {
			num, err := libseccomp.GetSyscallFromName(name)
			if err != nil {
				// Ignore unknown syscalls.
				if errors.Is(err, libseccomp.ErrSyscallDoesNotExist) {
					slog.Debug("unknown syscall", "name", name)
					continue
				}
				return nil, fmt.Errorf("get syscall from name: %w", err)
			}

			if len(sc.Args) == 0 {
				if err := filter.AddRule(num, action); err != nil {
					filter.Release()
					return nil, fmt.Errorf("add seccomp rule for %s: %w", name, err)
				}
				continue
			}

			var conditions []libseccomp.ScmpCondition

			for _, arg := range sc.Args {
				op := mapSeccompOperator(arg.Op)

				var cond libseccomp.ScmpCondition
				if arg.Op == specs.OpMaskedEqual {
					cond, err = libseccomp.MakeCondition(
						arg.Index,
						op,
						arg.ValueTwo,
						arg.Value,
					)
				} else {
					cond, err = libseccomp.MakeCondition(
						arg.Index,
						op,
						arg.Value,
						arg.ValueTwo,
					)
				}
				if err != nil {
					filter.Release()
					return nil, fmt.Errorf(
						"make seccomp condition for %s: %w",
						name,
						err,
					)
				}

				conditions = append(conditions, cond)
			}

			if err := filter.AddRuleConditional(num, action, conditions); err != nil {
				filter.Release()
				return nil, fmt.Errorf(
					"add seccomp conditional rule for %s: %w",
					name,
					err,
				)
			}
		}
	}

	// Workaround for when clone3 isn't explicitly allowed and default action is
	// EPERM, we add a rule to return ENOSYS instead to allow glibc to fallback
	// to clone.
	//
	// Also see:
	//  - https://github.com/youki-dev/youki/pull/2203/changes
	//  - https://github.com/moby/moby/pull/42681
	if spec.DefaultAction == specs.ActErrno {
		clone3Allowed := false
		for _, sc := range spec.Syscalls {
			if sc.Action == specs.ActAllow {
				if slices.Contains(sc.Names, "clone3") {
					clone3Allowed = true
				}
			}
			if clone3Allowed {
				break
			}
		}

		if !clone3Allowed {
			clone3, err := libseccomp.GetSyscallFromName("clone3")
			if err != nil {
				slog.Debug("clone3 syscall not found", "err", err)
			} else {
				slog.Debug("add clone3 ENOSYS workaround rule")
				enosys := libseccomp.ActErrno.SetReturnCode(int16(unix.ENOSYS))
				if err := filter.AddRule(clone3, enosys); err != nil {
					filter.Release()
					return nil, fmt.Errorf("add clone3 ENOSYS rule: %w", err)
				}
			}
		}
	}

	// Workaround for faccessat2 not being whitelisted in seccomp profile.
	//
	// Some seccomp profiles (specifically, the default OCI runtime-tools one)
	// only whitelist fsaccessat and not faccessat2. However glibc/musl
	// preferentially use fsaccessat2 if it's available (i.e. Linux 5.8+) and
	// don't fall back to fsaccessat in the case of EPERM.
	//
	// In the case the seccomp profile specifies fsaccessat but not fsaccesat2
	// we defensively add it.
	//
	// TODO: This feels a bit dodgy, especially given that other runtimes don't
	// appear to do anything similar. Need to re-review when more time.
	if spec.DefaultAction == specs.ActErrno {
		faccessatAllowed := false
		faccessat2Handled := false
		for _, sc := range spec.Syscalls {
			if slices.Contains(sc.Names, "faccessat") &&
				sc.Action == specs.ActAllow {
				faccessatAllowed = true
			}
			if slices.Contains(sc.Names, "faccessat2") {
				faccessat2Handled = true
			}
		}

		if faccessatAllowed && !faccessat2Handled {
			faccessat2, err := libseccomp.GetSyscallFromName("faccessat2")
			if err != nil {
				slog.Debug("faccessat2 syscall not found", "err", err)
			} else {
				slog.Debug("add faccessat2 allow rule (faccessat is allowed)")
				if err := filter.AddRule(faccessat2, libseccomp.ActAllow); err != nil {
					filter.Release()
					return nil, fmt.Errorf("add faccessat2 allow rule: %w", err)
				}
			}
		}
	}

	return filter, nil
}

func LoadSeccompFilter(spec *specs.LinuxSeccomp) error {
	filter, err := buildSeccompFilter(spec)
	if err != nil {
		return err
	}
	defer filter.Release()

	if err := filter.SetNoNewPrivsBit(false); err != nil {
		return fmt.Errorf("set seccomp no new privs bit: %w", err)
	}

	return filter.Load()
}
