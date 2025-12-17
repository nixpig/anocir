package cri

import (
	"context"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type criServer struct {
	runtimeapi.UnimplementedImageServiceServer
	runtimeapi.UnimplementedRuntimeServiceServer

	grpcServer *grpc.Server
	addr       string
	listener   net.Listener

	mu sync.Mutex
}

func newCRIServer(listener net.Listener) *criServer {
	return &criServer{listener: listener}
}

func (cs *criServer) start() error {
	cs.mu.Lock()

	cs.grpcServer = grpc.NewServer()
	cs.addr = cs.listener.Addr().String()

	runtimeapi.RegisterImageServiceServer(cs.grpcServer, cs)
	runtimeapi.RegisterRuntimeServiceServer(cs.grpcServer, cs)

	cs.mu.Unlock()

	return cs.grpcServer.Serve(cs.listener)
}

func (cs *criServer) shutdown() {
	cs.mu.Lock()
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
		grpcServer.Stop()
	}
}

func (cs *criServer) RunPodSandbox(
	ctx context.Context,
	req *runtimeapi.RunPodSandboxRequest,
) (*runtimeapi.RunPodSandboxResponse, error) {
	// config := req.GetConfig()
	return nil, nil
}
