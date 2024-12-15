package signal

import (
	"fmt"
	"syscall"
)

func FromInt(s int) (syscall.Signal, error) {
	switch s {
	case 1:
		return syscall.SIGHUP, nil
	case 2:
		return syscall.SIGINT, nil
	case 3:
		return syscall.SIGQUIT, nil
	case 4:
		return syscall.SIGILL, nil
	case 5:
		return syscall.SIGTRAP, nil
	case 6:
		return syscall.SIGIOT, nil
	case 7:
		return syscall.SIGBUS, nil
	case 8:
		return syscall.SIGFPE, nil
	case 9:
		return syscall.SIGKILL, nil
	case 10:
		return syscall.SIGUSR1, nil
	case 11:
		return syscall.SIGSEGV, nil
	case 12:
		return syscall.SIGUSR2, nil
	case 13:
		return syscall.SIGPIPE, nil
	case 14:
		return syscall.SIGALRM, nil
	case 15:
		return syscall.SIGTERM, nil
	case 16:
		return syscall.SIGSTKFLT, nil
	case 17:
		return syscall.SIGCHLD, nil
	case 18:
		return syscall.SIGCONT, nil
	case 19:
		return syscall.SIGSTOP, nil
	case 20:
		return syscall.SIGTSTP, nil
	case 21:
		return syscall.SIGTTIN, nil
	case 22:
		return syscall.SIGTTOU, nil
	case 23:
		return syscall.SIGURG, nil
	case 24:
		return syscall.SIGXCPU, nil
	case 25:
		return syscall.SIGXFSZ, nil
	case 26:
		return syscall.SIGVTALRM, nil
	case 27:
		return syscall.SIGPROF, nil
	case 28:
		return syscall.SIGWINCH, nil
	case 29:
		return syscall.SIGIO, nil
	case 30:
		return syscall.SIGPWR, nil
	}

	return 0, fmt.Errorf("signal not recognised (%d)", s)
}

func FromString(s string) (syscall.Signal, error) {
	switch s {
	case "SIGHUP", "HUP", "1":
		return syscall.SIGHUP, nil
	case "SIGINT", "INT", "2":
		return syscall.SIGINT, nil
	case "SIGQUIT", "QUIT", "3":
		return syscall.SIGQUIT, nil
	case "SIGILL", "ILL", "4":
		return syscall.SIGILL, nil
	case "SIGTRAP", "TRAP", "5":
		return syscall.SIGTRAP, nil
	case "SIGIOT", "IOT", "6":
		return syscall.SIGIOT, nil
	case "SIGBUS", "BUS", "7":
		return syscall.SIGBUS, nil
	case "SIGFPE", "FPE", "8":
		return syscall.SIGFPE, nil
	case "SIGKILL", "KILL", "9":
		return syscall.SIGKILL, nil
	case "SIGUSR1", "USR1", "10":
		return syscall.SIGUSR1, nil
	case "SIGSEGV", "SEGV", "11":
		return syscall.SIGSEGV, nil
	case "SIGUSR2", "USR2", "12":
		return syscall.SIGUSR2, nil
	case "SIGPIPE", "PIPE", "13":
		return syscall.SIGPIPE, nil
	case "SIGALRM", "ALRM", "14":
		return syscall.SIGALRM, nil
	case "SIGTERM", "TERM", "15":
		return syscall.SIGTERM, nil
	case "SIGSTKFLT", "STKFLT", "16":
		return syscall.SIGSTKFLT, nil
	case "SIGCHLD", "CHLD", "17":
		return syscall.SIGCHLD, nil
	case "SIGCONT", "CONT", "18":
		return syscall.SIGCONT, nil
	case "SIGSTOP", "STOP", "19":
		return syscall.SIGSTOP, nil
	case "SIGTSTP", "TSTP", "20":
		return syscall.SIGTSTP, nil
	case "SIGTTIN", "TTIN", "21":
		return syscall.SIGTTIN, nil
	case "SIGTTOU", "TTOU", "22":
		return syscall.SIGTTOU, nil
	case "SIGURG", "URG", "23":
		return syscall.SIGURG, nil
	case "SIGXCPU", "XCPU", "24":
		return syscall.SIGXCPU, nil
	case "SIGXFSZ", "XFSZ", "25":
		return syscall.SIGXFSZ, nil
	case "SIGVTALRM", "VTALRM", "26":
		return syscall.SIGVTALRM, nil
	case "SIGPROF", "PROF", "27":
		return syscall.SIGPROF, nil
	case "SIGWINCH", "WINCH", "28":
		return syscall.SIGWINCH, nil
	case "SIGIO", "IO", "29":
		return syscall.SIGIO, nil
	case "SIGPWR", "PWR", "30":
		return syscall.SIGPWR, nil
	}

	return syscall.Signal(0), fmt.Errorf("convert signal string to int (%s)", s)
}
