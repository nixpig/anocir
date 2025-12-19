package shim_test

import (
	"context"
	"testing"

	"github.com/containerd/containerd/api/types/runtimeoptions/v1"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/stretchr/testify/require"
)

const (
	// runtimeName       = "io.containerd.runc.v1"
	runtimeName       = "io.containerd.anocir.v0"
	testContainerName = "test-container"
	testSnapshotName  = "test-container-snapshot"
)

func TestShimCntrDIntegration(t *testing.T) {
	ctx, client := setupTestClient(t)

	// sudo ctr image pull docker.io/library/busybox:latest
	img, err := client.GetImage(ctx, "docker.io/library/busybox:latest")
	require.NoError(t, err)

	var opts runtimeoptions.Options

	cntr, err := client.NewContainer(
		ctx,
		"test-container",
		containerd.WithSnapshotter("overlayfs"),
		containerd.WithNewSnapshot(testSnapshotName, img),
		containerd.WithImage(img),
		containerd.WithNewSpec(oci.WithImageConfig(img)),
		containerd.WithRuntime(runtimeName, &opts),
	)
	require.NoError(t, err)

	defer func() {
		cntr.Delete(ctx, containerd.WithSnapshotCleanup)
		client.SnapshotService("overlayfs").Remove(ctx, testSnapshotName)
		client.Close()
	}()

	task, err := cntr.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	require.NoError(t, err)
	require.NotNil(t, task)
}

func setupTestClient(t *testing.T) (context.Context, *containerd.Client) {
	t.Helper()

	ctx := namespaces.WithNamespace(t.Context(), "default")

	client, err := containerd.New("/run/containerd/containerd.sock")
	require.NoError(t, err)

	if existing, err := client.LoadContainer(ctx, testContainerName); err == nil {
		existing.Delete(ctx, containerd.WithSnapshotCleanup)
	}

	client.SnapshotService("overlayfs").Remove(ctx, testSnapshotName)

	return ctx, client
}
