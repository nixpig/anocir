package shim

import (
	"context"
	"io"

	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/pkg/shim"
)

var _ shim.Manager = (*Manager)(nil)

type Manager struct {
	name string
}

func newManager(name string) *Manager {
	return &Manager{name}
}

func (m *Manager) Name() string {
	return m.name
}

func (m *Manager) Info(
	ctx context.Context,
	optionsR io.Reader,
) (*types.RuntimeInfo, error) {
	panic("unimplemented")
}

func (m *Manager) Start(
	ctx context.Context,
	id string,
	opts shim.StartOpts,
) (shim.BootstrapParams, error) {
	panic("unimplemented")
}

func (m *Manager) Stop(
	ctx context.Context,
	id string,
) (shim.StopStatus, error) {
	panic("unimplemented")
}
