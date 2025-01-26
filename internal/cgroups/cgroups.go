package cgroups

import (
	"github.com/containerd/cgroups/v3"
)

func IsUnified() bool {
	return cgroups.Mode() == cgroups.Unified
}
