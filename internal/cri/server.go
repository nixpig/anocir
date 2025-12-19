package cri

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type criServer struct {
	runtimeapi.UnimplementedImageServiceServer
	runtimeapi.UnimplementedRuntimeServiceServer

	grpcServer *grpc.Server
	listener   net.Listener
	logger     *slog.Logger

	mu sync.Mutex
}

func newCRIServer(listener net.Listener, logger *slog.Logger) *criServer {
	return &criServer{listener: listener, logger: logger}
}

func (cs *criServer) start() error {
	cs.mu.Lock()

	cs.logger.Info("starting server", "addr", cs.listener.Addr().String())

	grpcServer := grpc.NewServer()

	runtimeapi.RegisterImageServiceServer(grpcServer, cs)
	runtimeapi.RegisterRuntimeServiceServer(grpcServer, cs)

	cs.grpcServer = grpcServer

	cs.mu.Unlock()

	return grpcServer.Serve(cs.listener)
}

func (cs *criServer) shutdown() {
	cs.mu.Lock()

	cs.logger.Info("shutting down server")

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
		cs.logger.Info("graceful shutdown timed out, terminating")
		grpcServer.Stop()
	}
}

func (cs *criServer) Version(
	ctx context.Context,
	req *runtimeapi.VersionRequest,
) (*runtimeapi.VersionResponse, error) {
	cs.logger.Debug("Version", "version", req.GetVersion())

	return &runtimeapi.VersionResponse{
		Version:           "0.35.0",
		RuntimeName:       "anocir",
		RuntimeVersion:    "0.0.4",
		RuntimeApiVersion: "0.35.0",
	}, nil
}

func (cs *criServer) ImageFsInfo(
	ctx context.Context,
	req *runtimeapi.ImageFsInfoRequest,
) (*runtimeapi.ImageFsInfoResponse, error) {
	cs.logger.Debug("ImageFsInfo", "req", req.String())

	return &runtimeapi.ImageFsInfoResponse{
		ImageFilesystems:     []*runtimeapi.FilesystemUsage{},
		ContainerFilesystems: []*runtimeapi.FilesystemUsage{},
	}, nil
}

func (cs *criServer) RunPodSandbox(
	ctx context.Context,
	req *runtimeapi.RunPodSandboxRequest,
) (*runtimeapi.RunPodSandboxResponse, error) {
	config := req.GetConfig()
	runtimeHandler := req.GetRuntimeHandler()

	cs.logger.Debug("RunPodSandbox", "config", config)
	cs.logger.Debug("RunPodSandbox", "runtime_handler", runtimeHandler)

	return &runtimeapi.RunPodSandboxResponse{
		PodSandboxId: uuid.NewString(),
	}, nil
}

func (cs *criServer) PodSandboxStatus(
	ctx context.Context,
	req *runtimeapi.PodSandboxStatusRequest,
) (*runtimeapi.PodSandboxStatusResponse, error) {
	return &runtimeapi.PodSandboxStatusResponse{
		Status:    &runtimeapi.PodSandboxStatus{},
		Info:      map[string]string{},
		Timestamp: time.Now().Unix(),
	}, nil
}
