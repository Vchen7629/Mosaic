//go:build integration

package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/test"
)

// fastRetry is a retry config with short backoffs so tests run quickly.
var fastRetry = handler.RetryConfig{
	MaxAttempts:       3,
	InitialBackoff:    20 * time.Millisecond,
	MaxBackoff:        100 * time.Millisecond,
	BackoffMultiplier: 2.0,
	RetryableCodes:    []codes.Code{codes.Unavailable},
}

func TestRetry(t *testing.T) {
	t.Run("attempt count and final error", func(t *testing.T) {
		tests := []struct {
			name         string
			config       handler.RetryConfig
			responseCode codes.Code
			wantAttempts int32
			wantCode     codes.Code
		}{
			{
				name:         "Unavailable exhausts MaxAttempts",
				config:       fastRetry,
				responseCode: codes.Unavailable,
				wantAttempts: int32(fastRetry.MaxAttempts),
				wantCode:     codes.Unavailable,
			},
			{
				name:         "non-retryable NotFound returns on first attempt",
				config:       fastRetry,
				responseCode: codes.NotFound,
				wantAttempts: 1,
				wantCode:     codes.NotFound,
			},
			{
				name:         "NoRetry config makes exactly one attempt",
				config:       handler.NoRetry,
				responseCode: codes.Unavailable,
				wantAttempts: 1,
				wantCode:     codes.Unavailable,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				var attempt atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					attempt.Add(1)
					w.Header().Set("Content-Type", "application/grpc-web+proto")
					w.WriteHeader(http.StatusOK)
					w.Write(test.GrpcTrailerFrame(tc.responseCode, "error")) //nolint:errcheck
				}))
				defer srv.Close()

				client := handler.NewClient(srv.URL, tc.config)
				var reply fd.ProcessVisitorFacesResponse

				err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

				test.RequireGRPCCode(t, err, tc.wantCode)
				assert.Equal(t, tc.wantAttempts, attempt.Load())
			})
		}
	})

	t.Run("succeeds once server recovers after two Unavailable responses", func(t *testing.T) {
		var attempt atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := attempt.Add(1)
			w.Header().Set("Content-Type", "application/grpc-web+proto")
			w.WriteHeader(http.StatusOK)
			if n < 3 {
				w.Write(test.GrpcTrailerFrame(codes.Unavailable, "not ready")) //nolint:errcheck
			} else {
				w.Write(test.GrpcDataFrame(t, &fd.ProcessVisitorFacesResponse{FaceDetected: true})) //nolint:errcheck
			}
		}))
		defer srv.Close()

		client := handler.NewClient(srv.URL, fastRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

		require.NoError(t, err)
		assert.True(t, reply.FaceDetected)
		assert.Equal(t, int32(3), attempt.Load())
	})

	t.Run("context cancelled during backoff returns before next attempt", func(t *testing.T) {
		slowRetry := handler.RetryConfig{
			MaxAttempts:       3,
			InitialBackoff:    300 * time.Millisecond,
			MaxBackoff:        1 * time.Second,
			BackoffMultiplier: 2.0,
			RetryableCodes:    []codes.Code{codes.Unavailable},
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/grpc-web+proto")
			w.WriteHeader(http.StatusOK)
			w.Write(test.GrpcTrailerFrame(codes.Unavailable, "not ready")) //nolint:errcheck
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(80 * time.Millisecond) // cancel mid-way through the 300ms backoff
			cancel()
		}()

		client := handler.NewClient(srv.URL, slowRetry)
		var reply fd.ProcessVisitorFacesResponse

		start := time.Now()
		err := client.Invoke(ctx, "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

		require.Error(t, err)
		assert.Less(t, time.Since(start), 250*time.Millisecond, "context cancel must short-circuit the backoff")
	})
}
