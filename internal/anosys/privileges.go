package anosys

import (
	"golang.org/x/sys/unix"
)

// SetNoNewPrivs sets the PR_SET_NO_NEW_PRIVS flag for the current (container)
// process, preventing it from gaining new privileges.
func SetNoNewPrivs() error {
	return unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)
}
