package cri

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type criServer struct {
	runtimeapi.UnimplementedImageServiceServer
	runtimeapi.UnimplementedRuntimeServiceServer

	grpcServer *grpc.Server
	listener   net.Listener
	log        *slog.Logger

	mu sync.Mutex
}

func newCRIServer(listener net.Listener, logger *slog.Logger) *criServer {
	return &criServer{listener: listener, log: logger}
}

func (cs *criServer) start() error {
	cs.mu.Lock()

	cs.log.Info("starting server", "addr", cs.listener.Addr().String())

	grpcServer := grpc.NewServer()

	runtimeapi.RegisterImageServiceServer(grpcServer, cs)
	runtimeapi.RegisterRuntimeServiceServer(grpcServer, cs)

	cs.grpcServer = grpcServer

	cs.mu.Unlock()

	return grpcServer.Serve(cs.listener)
}

func (cs *criServer) shutdown() {
	cs.mu.Lock()

	cs.log.Info("shutting down server")

	grpcServer := cs.grpcServer

	cs.mu.Unlock()

	if grpcServer == nil {
		return
	}

	doneCh := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		cs.log.Info("graceful shutdown timed out, terminating")
		grpcServer.Stop()
	}
}

func (cs *criServer) RunPodSandbox(
	ctx context.Context,
	req *runtimeapi.RunPodSandboxRequest,
) (*runtimeapi.RunPodSandboxResponse, error) {
	cs.log.Info("request: RunPodSandbox")
	// config := req.GetConfig()
	return nil, status.Error(
		codes.Unimplemented,
		"RunPodSandbox not implemented",
	)
}
