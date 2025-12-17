package cri

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func TestCRIServer_Basic(t *testing.T) {
	client := setupTestServerAndClient(t)

	resp, err := client.RunPodSandbox(
		t.Context(),
		&runtimeapi.RunPodSandboxRequest{},
	)
	assert.NoError(t, err)
	assert.Equal(t, "", resp.PodSandboxId)
}

func setupTestServerAndClient(t *testing.T) runtimeapi.RuntimeServiceClient {
	t.Helper()

	socket := filepath.Join(t.TempDir(), "test.sock")

	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)

	server := newCRIServer(listener)

	go func() {
		err := server.start()
		require.NoError(t, err)
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
