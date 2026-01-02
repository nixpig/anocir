package container

import (
	"fmt"
)

type ExecOpts struct {
	ContainerPID int
	Rootfs       string
	Cwd          string
	Args         []string

	ConsoleSocket  string
	UID            int
	PIDFile        string
	TTY            bool
	Detach         bool
	IgnorePaused   bool
	PreserveFDs    int
	Env            map[string]string
	AdditionalGIDs []int
	Process        string
	ProcessLabel   string
	AppArmor       string
	NoNewPrivs     bool
	Capabilities   []string
	Cgroup         string
}

func Exec(opts *ExecOpts) error {
	// setns, fork, execve
	fmt.Printf("%+v\n", opts)

	// 1. Reads container state to get the init PID
	// 2. Opens /proc/<init-pid>/ns/* for each namespace
	// 3. Calls setns for each (mount ns via C constructor like you do now)
	// 4. Forks (or the setns caller is the final process if no PID ns join needed)
	// 5. Child does minimal setup (cwd, uid/gid, caps, seccomp) then execve

	return nil
}
