//go:build integration

package main

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
	fd "mosaic-face-detection.com/gen"
)

// connTracker counts active server-side connections via gRPC stats hooks
type connTracker struct{ active atomic.Int32 }

func (t *connTracker) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context { return ctx }
func (t *connTracker) HandleRPC(_ context.Context, _ stats.RPCStats)                   {}
func (t *connTracker) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}
func (t *connTracker) HandleConn(_ context.Context, s stats.ConnStats) {
	switch s.(type) {
	case *stats.ConnBegin:
		t.active.Add(1)
	case *stats.ConnEnd:
		t.active.Add(-1)
	}
}

func TestStaleConnectionsClosed(t *testing.T) {
	const idleTimeout = 300 * time.Millisecond

	tracker := &connTracker{}

	cfg := &Config{MaxConnectionIdle: idleTimeout}
	opts := append(cfg.serverOptions(), grpc.StatsHandler(tracker))
	srv := grpc.NewServer(opts...)
	fd.RegisterFaceDetectionServiceServer(srv, &fd.UnimplementedFaceDetectionServiceServer{})

	lis, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	go srv.Serve(lis) //nolint:errcheck
	defer srv.Stop()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	defer conn.Close()

	client := fd.NewFaceDetectionServiceClient(conn)
	client.SyncProfile(context.Background(), &fd.SyncProfileRequest{}) //nolint:errcheck

	time.Sleep(idleTimeout + 200*time.Millisecond)

	assert.Eventually(t, func() bool {
		return tracker.active.Load() == 0
	}, time.Second, 10*time.Millisecond, "stale idle connection should be closed by server")
}
