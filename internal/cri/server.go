package cri

import (
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type CRIServer struct {
	runtimeapi.UnimplementedImageServiceServer
	runtimeapi.UnimplementedRuntimeServiceServer

	grpcServer *grpc.Server
	addr       string

	mu sync.Mutex
}

func NewCRIServer() *CRIServer {
	return &CRIServer{}
}

func (cs *CRIServer) Start(listener net.Listener) error {
	cs.mu.Lock()
	cs.grpcServer = grpc.NewServer()
	cs.addr = listener.Addr().String()
	cs.mu.Unlock()

	runtimeapi.RegisterImageServiceServer(cs.grpcServer, cs)
	runtimeapi.RegisterRuntimeServiceServer(cs.grpcServer, cs)

	return cs.grpcServer.Serve(listener)
}

func (cs *CRIServer) Shutdown() {
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
