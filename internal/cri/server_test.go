package cri

import (
	"io"
	"net"
	"path/filepath"
	"testing"

	"github.com/nixpig/anocir/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func TestCRIServer_Basic(t *testing.T) {
	client := setupTestServerAndClient(t)

	_, err := client.RunPodSandbox(
		t.Context(),
		&runtimeapi.RunPodSandboxRequest{},
	)

	assertGRPCStatus(t, err, codes.Unimplemented)
}

func setupTestServerAndClient(t *testing.T) runtimeapi.RuntimeServiceClient {
	t.Helper()

	socket := filepath.Join(t.TempDir(), "test.sock")

	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)

	server := newCRIServer(listener, logging.NewLogger(io.Discard, false))

	go func() {
		_ = server.start()
	}()

	t.Cleanup(func() {
		server.shutdown()
	})

	clientConn, err := grpc.NewClient(
		"unix:"+socket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		clientConn.Close()
	})

	client := runtimeapi.NewRuntimeServiceClient(clientConn)

	return client
}

func assertGRPCStatus(t *testing.T, err error, want codes.Code) {
	t.Helper()

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, want, st.Code())
}
