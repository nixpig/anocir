package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func PivotRoot(containerRootfs string) error {
	fmt.Println("Pivot container root filesystem")
	oldroot := filepath.Join(containerRootfs, ".oldroot")

	if err := syscall.Mount(
		containerRootfs,
		containerRootfs,
		"",
		syscall.MS_BIND|syscall.MS_REC,
		"",
	); err != nil {
		return err
	}

	if err := os.MkdirAll(oldroot, 0700); err != nil {
		return err
	}

	if err := syscall.PivotRoot(containerRootfs, oldroot); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	if err := syscall.Unmount(".oldroot", syscall.MNT_DETACH); err != nil {
		return err
	}

	if err := os.RemoveAll(".oldroot"); err != nil {
		return err
	}

	return nil
}
