package platform

import "golang.org/x/sys/unix"

func SetWinSize(fd uintptr, width, height uint) error {
	return unix.IoctlSetWinsize(
		int(fd),
		unix.TIOCSWINSZ,
		&unix.Winsize{
			Col: uint16(width),
			Row: uint16(height),
		},
	)
}
