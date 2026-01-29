package platform

import (
	"errors"

	"golang.org/x/sys/unix"
)

var ErrUnknownSignal = errors.New("unknown signal")

// SendSignal sends a signal to the specified process ID.
func SendSignal(pid int, sig unix.Signal) error {
	return unix.Kill(pid, sig)
}

// ParseSignal parses the given sig and returns the corresponding a unix.Signal.
// If the signal is not recognised then unix.Signal(0) is returned.
func ParseSignal(sig string) (unix.Signal, error) {
	switch sig {
	case "SIGHUP", "HUP", "1":
		return unix.SIGHUP, nil
	case "SIGINT", "INT", "2":
		return unix.SIGINT, nil
	case "SIGQUIT", "QUIT", "3":
		return unix.SIGQUIT, nil
	case "SIGILL", "ILL", "4":
		return unix.SIGILL, nil
	case "SIGTRAP", "TRAP", "5":
		return unix.SIGTRAP, nil
	case "SIGIOT", "IOT", "6":
		return unix.SIGIOT, nil
	case "SIGBUS", "BUS", "7":
		return unix.SIGBUS, nil
	case "SIGFPE", "FPE", "8":
		return unix.SIGFPE, nil
	case "SIGKILL", "KILL", "9":
		return unix.SIGKILL, nil
	case "SIGUSR1", "USR1", "10":
		return unix.SIGUSR1, nil
	case "SIGSEGV", "SEGV", "11":
		return unix.SIGSEGV, nil
	case "SIGUSR2", "USR2", "12":
		return unix.SIGUSR2, nil
	case "SIGPIPE", "PIPE", "13":
		return unix.SIGPIPE, nil
	case "SIGALRM", "ALRM", "14":
		return unix.SIGALRM, nil
	case "SIGTERM", "TERM", "15":
		return unix.SIGTERM, nil
	case "SIGSTKFLT", "STKFLT", "16":
		return unix.SIGSTKFLT, nil
	case "SIGCHLD", "CHLD", "17":
		return unix.SIGCHLD, nil
	case "SIGCONT", "CONT", "18":
		return unix.SIGCONT, nil
	case "SIGSTOP", "STOP", "19":
		return unix.SIGSTOP, nil
	case "SIGTSTP", "TSTP", "20":
		return unix.SIGTSTP, nil
	case "SIGTTIN", "TTIN", "21":
		return unix.SIGTTIN, nil
	case "SIGTTOU", "TTOU", "22":
		return unix.SIGTTOU, nil
	case "SIGURG", "URG", "23":
		return unix.SIGURG, nil
	case "SIGXCPU", "XCPU", "24":
		return unix.SIGXCPU, nil
	case "SIGXFSZ", "XFSZ", "25":
		return unix.SIGXFSZ, nil
	case "SIGVTALRM", "VTALRM", "26":
		return unix.SIGVTALRM, nil
	case "SIGPROF", "PROF", "27":
		return unix.SIGPROF, nil
	case "SIGWINCH", "WINCH", "28":
		return unix.SIGWINCH, nil
	case "SIGIO", "IO", "29":
		return unix.SIGIO, nil
	case "SIGPWR", "PWR", "30":
		return unix.SIGPWR, nil
	}

	return unix.Signal(0), ErrUnknownSignal
}
