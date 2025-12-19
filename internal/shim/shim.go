package shim

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/shim"
)

const shimName = "io.containerd.anocir.v0"

func Run() {
	ctx := context.Background()

	registerTaskPlugin()

	shim.Run(ctx, newManager(shimName))
}
