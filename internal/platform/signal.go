package platform

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// SendSignal sends a signal to the specified process ID.
func SendSignal(pid int, sig unix.Signal) error {
	return syscall.Kill(pid, sig)
}

// ParseSignal parses the given sig and returns the corresponding a unix.Signal.
// If the signal is not recognised then unix.Signal(0) is returned.
func ParseSignal(sig string) unix.Signal {
	switch sig {
	case "SIGHUP", "HUP", "1":
		return unix.SIGHUP
	case "SIGINT", "INT", "2":
		return unix.SIGINT
	case "SIGQUIT", "QUIT", "3":
		return unix.SIGQUIT
	case "SIGILL", "ILL", "4":
		return unix.SIGILL
	case "SIGTRAP", "TRAP", "5":
		return unix.SIGTRAP
	case "SIGIOT", "IOT", "6":
		return unix.SIGIOT
	case "SIGBUS", "BUS", "7":
		return unix.SIGBUS
	case "SIGFPE", "FPE", "8":
		return unix.SIGFPE
	case "SIGKILL", "KILL", "9":
		return unix.SIGKILL
	case "SIGUSR1", "USR1", "10":
		return unix.SIGUSR1
	case "SIGSEGV", "SEGV", "11":
		return unix.SIGSEGV
	case "SIGUSR2", "USR2", "12":
		return unix.SIGUSR2
	case "SIGPIPE", "PIPE", "13":
		return unix.SIGPIPE
	case "SIGALRM", "ALRM", "14":
		return unix.SIGALRM
	case "SIGTERM", "TERM", "15":
		return unix.SIGTERM
	case "SIGSTKFLT", "STKFLT", "16":
		return unix.SIGSTKFLT
	case "SIGCHLD", "CHLD", "17":
		return unix.SIGCHLD
	case "SIGCONT", "CONT", "18":
		return unix.SIGCONT
	case "SIGSTOP", "STOP", "19":
		return unix.SIGSTOP
	case "SIGTSTP", "TSTP", "20":
		return unix.SIGTSTP
	case "SIGTTIN", "TTIN", "21":
		return unix.SIGTTIN
	case "SIGTTOU", "TTOU", "22":
		return unix.SIGTTOU
	case "SIGURG", "URG", "23":
		return unix.SIGURG
	case "SIGXCPU", "XCPU", "24":
		return unix.SIGXCPU
	case "SIGXFSZ", "XFSZ", "25":
		return unix.SIGXFSZ
	case "SIGVTALRM", "VTALRM", "26":
		return unix.SIGVTALRM
	case "SIGPROF", "PROF", "27":
		return unix.SIGPROF
	case "SIGWINCH", "WINCH", "28":
		return unix.SIGWINCH
	case "SIGIO", "IO", "29":
		return unix.SIGIO
	case "SIGPWR", "PWR", "30":
		return unix.SIGPWR
	}

	return unix.Signal(0)
}
