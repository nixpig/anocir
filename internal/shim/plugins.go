package shim

import (
	"fmt"

	"github.com/containerd/containerd/pkg/shutdown"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/containerd/v2/plugins"
	"github.com/containerd/plugin"
	"github.com/containerd/plugin/registry"
)

func registerTaskPlugin() {
	registry.Register(&plugin.Registration{
		Type: plugins.TTRPCPlugin,
		ID:   "task",
		Requires: []plugin.Type{
			plugins.EventPlugin,
			plugins.InternalPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (any, error) {
			sp, err := ic.GetByID(plugins.EventPlugin, "publisher")
			if err != nil {
				return nil, fmt.Errorf("get publisher plugin: %s", err)
			}

			ss, err := ic.GetByID(plugins.InternalPlugin, "shutdown")
			if err != nil {
				return nil, fmt.Errorf("get shutdown plugin: %w", err)
			}

			return newTaskService(
				ic.Context,
				sp.(shim.Publisher),
				ss.(shutdown.Service),
			)
		},
	})
}
