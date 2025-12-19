package shim

import (
	"context"
	"fmt"
	"sync"

	api "github.com/containerd/containerd/api/runtime/task/v2"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/containerd/v2/pkg/shutdown"
	"github.com/containerd/ttrpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	_ api.TaskService   = (*taskService)(nil)
	_ shim.TTRPCService = (*taskService)(nil)
)

type taskService struct {
	mu sync.Mutex

	sp shim.Publisher
	ss shutdown.Service
}

func newTaskService(
	ctx context.Context,
	sp shim.Publisher,
	ss shutdown.Service,
) (api.TaskService, error) {
	// TODO: Is "address" correct?
	sockAddr, err := shim.ReadAddress("address")
	if err != nil {
		return nil, fmt.Errorf("read socket address: %w", err)
	}

	// Remove socket on shutdown.
	ss.RegisterCallback(func(ctx context.Context) error {
		if err := shim.RemoveSocket(sockAddr); err != nil {
			return fmt.Errorf("remove shim socket: %w", err)
		}

		return nil
	})

	return &taskService{sp: sp, ss: ss}, nil
}

// RegisterTTRPC implements shim.TTRPCService.
func (t *taskService) RegisterTTRPC(*ttrpc.Server) error {
	panic("unimplemented")
}

// Checkpoint implements task.TaskService.
func (t *taskService) Checkpoint(
	context.Context,
	*api.CheckpointTaskRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// CloseIO implements task.TaskService.
func (t *taskService) CloseIO(
	context.Context,
	*api.CloseIORequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Connect implements task.TaskService.
func (t *taskService) Connect(
	context.Context,
	*api.ConnectRequest,
) (*api.ConnectResponse, error) {
	panic("unimplemented")
}

// Create implements task.TaskService.
func (t *taskService) Create(
	context.Context,
	*api.CreateTaskRequest,
) (*api.CreateTaskResponse, error) {
	panic("unimplemented")
}

// Delete implements task.TaskService.
func (t *taskService) Delete(
	context.Context,
	*api.DeleteRequest,
) (*api.DeleteResponse, error) {
	panic("unimplemented")
}

// Exec implements task.TaskService.
func (t *taskService) Exec(
	context.Context,
	*api.ExecProcessRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Kill implements task.TaskService.
func (t *taskService) Kill(
	context.Context,
	*api.KillRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Pause implements task.TaskService.
func (t *taskService) Pause(
	context.Context,
	*api.PauseRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Pids implements task.TaskService.
func (t *taskService) Pids(
	context.Context,
	*api.PidsRequest,
) (*api.PidsResponse, error) {
	panic("unimplemented")
}

// ResizePty implements task.TaskService.
func (t *taskService) ResizePty(
	context.Context,
	*api.ResizePtyRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Resume implements task.TaskService.
func (t *taskService) Resume(
	context.Context,
	*api.ResumeRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Shutdown implements task.TaskService.
func (t *taskService) Shutdown(
	context.Context,
	*api.ShutdownRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Start implements task.TaskService.
func (t *taskService) Start(
	context.Context,
	*api.StartRequest,
) (*api.StartResponse, error) {
	panic("unimplemented")
}

// State implements task.TaskService.
func (t *taskService) State(
	context.Context,
	*api.StateRequest,
) (*api.StateResponse, error) {
	panic("unimplemented")
}

// Stats implements task.TaskService.
func (t *taskService) Stats(
	context.Context,
	*api.StatsRequest,
) (*api.StatsResponse, error) {
	panic("unimplemented")
}

// Update implements task.TaskService.
func (t *taskService) Update(
	context.Context,
	*api.UpdateTaskRequest,
) (*emptypb.Empty, error) {
	panic("unimplemented")
}

// Wait implements task.TaskService.
func (t *taskService) Wait(
	context.Context,
	*api.WaitRequest,
) (*api.WaitResponse, error) {
	panic("unimplemented")
}
