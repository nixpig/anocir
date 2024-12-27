package signal

import (
	"syscall"
)

func FromInt(s int) syscall.Signal {
	switch s {
	case 1:
		return syscall.SIGHUP
	case 2:
		return syscall.SIGINT
	case 3:
		return syscall.SIGQUIT
	case 4:
		return syscall.SIGILL
	case 5:
		return syscall.SIGTRAP
	case 6:
		return syscall.SIGIOT
	case 7:
		return syscall.SIGBUS
	case 8:
		return syscall.SIGFPE
	case 9:
		return syscall.SIGKILL
	case 10:
		return syscall.SIGUSR1
	case 11:
		return syscall.SIGSEGV
	case 12:
		return syscall.SIGUSR2
	case 13:
		return syscall.SIGPIPE
	case 14:
		return syscall.SIGALRM
	case 15:
		return syscall.SIGTERM
	case 16:
		return syscall.SIGSTKFLT
	case 17:
		return syscall.SIGCHLD
	case 18:
		return syscall.SIGCONT
	case 19:
		return syscall.SIGSTOP
	case 20:
		return syscall.SIGTSTP
	case 21:
		return syscall.SIGTTIN
	case 22:
		return syscall.SIGTTOU
	case 23:
		return syscall.SIGURG
	case 24:
		return syscall.SIGXCPU
	case 25:
		return syscall.SIGXFSZ
	case 26:
		return syscall.SIGVTALRM
	case 27:
		return syscall.SIGPROF
	case 28:
		return syscall.SIGWINCH
	case 29:
		return syscall.SIGIO
	case 30:
		return syscall.SIGPWR
	}

	return 0
}

func FromString(s string) syscall.Signal {
	switch s {
	case "SIGHUP", "HUP", "1":
		return syscall.SIGHUP
	case "SIGINT", "INT", "2":
		return syscall.SIGINT
	case "SIGQUIT", "QUIT", "3":
		return syscall.SIGQUIT
	case "SIGILL", "ILL", "4":
		return syscall.SIGILL
	case "SIGTRAP", "TRAP", "5":
		return syscall.SIGTRAP
	case "SIGIOT", "IOT", "6":
		return syscall.SIGIOT
	case "SIGBUS", "BUS", "7":
		return syscall.SIGBUS
	case "SIGFPE", "FPE", "8":
		return syscall.SIGFPE
	case "SIGKILL", "KILL", "9":
		return syscall.SIGKILL
	case "SIGUSR1", "USR1", "10":
		return syscall.SIGUSR1
	case "SIGSEGV", "SEGV", "11":
		return syscall.SIGSEGV
	case "SIGUSR2", "USR2", "12":
		return syscall.SIGUSR2
	case "SIGPIPE", "PIPE", "13":
		return syscall.SIGPIPE
	case "SIGALRM", "ALRM", "14":
		return syscall.SIGALRM
	case "SIGTERM", "TERM", "15":
		return syscall.SIGTERM
	case "SIGSTKFLT", "STKFLT", "16":
		return syscall.SIGSTKFLT
	case "SIGCHLD", "CHLD", "17":
		return syscall.SIGCHLD
	case "SIGCONT", "CONT", "18":
		return syscall.SIGCONT
	case "SIGSTOP", "STOP", "19":
		return syscall.SIGSTOP
	case "SIGTSTP", "TSTP", "20":
		return syscall.SIGTSTP
	case "SIGTTIN", "TTIN", "21":
		return syscall.SIGTTIN
	case "SIGTTOU", "TTOU", "22":
		return syscall.SIGTTOU
	case "SIGURG", "URG", "23":
		return syscall.SIGURG
	case "SIGXCPU", "XCPU", "24":
		return syscall.SIGXCPU
	case "SIGXFSZ", "XFSZ", "25":
		return syscall.SIGXFSZ
	case "SIGVTALRM", "VTALRM", "26":
		return syscall.SIGVTALRM
	case "SIGPROF", "PROF", "27":
		return syscall.SIGPROF
	case "SIGWINCH", "WINCH", "28":
		return syscall.SIGWINCH
	case "SIGIO", "IO", "29":
		return syscall.SIGIO
	case "SIGPWR", "PWR", "30":
		return syscall.SIGPWR
	}

	return syscall.Signal(0)
}
