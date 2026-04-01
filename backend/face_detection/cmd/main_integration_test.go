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
	"google.golang.org/grpc/keepalive"
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

	srv := grpc.NewServer(
		grpc.StatsHandler(tracker),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: idleTimeout,
			Time:              100 * time.Millisecond,
			Timeout:           50 * time.Millisecond,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Millisecond,
			PermitWithoutStream: true,
		}),
	)
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

	// trigger connection establishment (RPC will fail as unimplemented, that's fine)
	client := fd.NewFaceDetectionServiceClient(conn)
	client.SyncProfile(context.Background(), &fd.SyncProfileRequest{}) //nolint:errcheck

	// wait for idle timeout + buffer, then verify stale connection was closed
	time.Sleep(idleTimeout + 200*time.Millisecond)

	assert.Eventually(t, func() bool {
		return tracker.active.Load() == 0
	}, time.Second, 10*time.Millisecond, "stale idle connection should be closed by server")
}
