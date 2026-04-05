//go:build unit

package handler_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	fd "mosaic-client.com/gen/face_detection"
	"mosaic-client.com/internal/handler"
	"mosaic-client.com/internal/test"
)

func TestInvoke(t *testing.T) {
	t.Run("response error paths", func(t *testing.T) {
		truncatedBodyFrame := func() []byte {
			var buf bytes.Buffer
			buf.WriteByte(0x00)
			length := make([]byte, 4)
			binary.BigEndian.PutUint32(length, 100)
			buf.Write(length)
			buf.Write([]byte{0x01, 0x02, 0x03}) // only 3 bytes of the declared 100
			return buf.Bytes()
		}

		tests := []struct {
			name     string
			body     []byte
			reply    interface{} // nil → uses *fd.ProcessVisitorFacesResponse
			wantCode codes.Code
			wantMsg  string
		}{
			{
				name:     "truncated frame header",
				body:     []byte{0x00, 0x00, 0x01}, // 3 bytes, need 5
				wantCode: codes.Internal,
				wantMsg:  "truncated frame header",
			},
			{
				name:     "truncated frame body",
				body:     truncatedBodyFrame(),
				wantCode: codes.Internal,
				wantMsg:  "truncated frame body",
			},
			{
				name:     "empty body - no data frame",
				body:     []byte{},
				wantCode: codes.Internal,
				wantMsg:  "no data frame in response",
			},
			{
				name:     "OK trailer only - no data frame",
				body:     test.GrpcTrailerFrame(codes.OK, ""),
				wantCode: codes.Internal,
				wantMsg:  "no data frame in response",
			},
			{
				name:     "non-OK gRPC trailer",
				body:     test.GrpcTrailerFrame(codes.NotFound, "record not found"),
				wantCode: codes.NotFound,
				wantMsg:  "record not found",
			},
			{
				name:     "data frame + non-OK trailer - trailer wins",
				body:     append(test.GrpcDataFrame(t, &fd.ProcessVisitorFacesResponse{FaceDetected: true}), test.GrpcTrailerFrame(codes.Unavailable, "overloaded")...),
				wantCode: codes.Unavailable,
				wantMsg:  "overloaded",
			},
			{
				name:     "non-proto reply type",
				body:     test.GrpcDataFrame(t, &fd.ProcessVisitorFacesResponse{}),
				reply:    new(string),
				wantCode: codes.Internal,
				wantMsg:  "reply must be proto.Message",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				srv := test.ServeGRPCWeb(tc.body)
				defer srv.Close()

				client := handler.NewClient(srv.URL, handler.NoRetry)
				reply := tc.reply
				if reply == nil {
					reply = &fd.ProcessVisitorFacesResponse{}
				}

				err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, reply)

				test.RequireGRPCCode(t, err, tc.wantCode)
				s, _ := status.FromError(err)
				assert.Equal(t, tc.wantMsg, s.Message())
			})
		}
	})

	t.Run("data frame only - reply is populated", func(t *testing.T) {
		srv := test.ServeGRPCWeb(test.GrpcDataFrame(t, &fd.ProcessVisitorFacesResponse{FaceDetected: true}))
		defer srv.Close()

		client := handler.NewClient(srv.URL, handler.NoRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

		require.NoError(t, err)
		assert.True(t, reply.FaceDetected)
	})

	t.Run("data frame + OK trailer - still succeeds", func(t *testing.T) {
		body := append(test.GrpcDataFrame(t, &fd.ProcessVisitorFacesResponse{FaceDetected: true}), test.GrpcTrailerFrame(codes.OK, "")...)
		srv := test.ServeGRPCWeb(body)
		defer srv.Close()

		client := handler.NewClient(srv.URL, handler.NoRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

		require.NoError(t, err)
		assert.True(t, reply.FaceDetected)
	})

	t.Run("trailing slash on baseURL is stripped", func(t *testing.T) {
		srv := test.ServeGRPCWeb(test.GrpcDataFrame(t, &fd.ProcessVisitorFacesResponse{}))
		defer srv.Close()

		client := handler.NewClient(srv.URL+"/", handler.NoRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)
		require.NoError(t, err)
	})

	t.Run("non-proto args - returns Internal without hitting server", func(t *testing.T) {
		client := handler.NewClient("http://localhost:9999", handler.NoRetry)

		err := client.Invoke(context.Background(), "/any/Method", "not-a-proto", &fd.ProcessVisitorFacesResponse{})

		test.RequireGRPCCode(t, err, codes.Internal)
		assert.ErrorContains(t, err, "args must be proto.Message")
	})

	t.Run("HTTP 500 - returns Internal", func(t *testing.T) {
		srv := test.ServeGRPCWebStatus(http.StatusInternalServerError)
		defer srv.Close()

		client := handler.NewClient(srv.URL, handler.NoRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

		test.RequireGRPCCode(t, err, codes.Internal)
		assert.ErrorContains(t, err, "http status: 500")
	})

	t.Run("server down - returns Unavailable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		srv.Close()

		client := handler.NewClient(srv.URL, handler.NoRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(context.Background(), "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)

		test.RequireGRPCCode(t, err, codes.Unavailable)
	})

	t.Run("cancelled context - returns error", func(t *testing.T) {
		srv := test.ServeGRPCWeb(nil)
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		client := handler.NewClient(srv.URL, handler.NoRetry)
		var reply fd.ProcessVisitorFacesResponse

		err := client.Invoke(ctx, "/any/Method", &fd.ProcessVisitorFacesRequest{}, &reply)
		require.Error(t, err)
	})
}

func TestNewStreamReturn(t *testing.T) {
	client := handler.NewClient("http://localhost:9999", handler.NoRetry)

	stream, err := client.NewStream(context.Background(), nil, "/any/Method")

	assert.Nil(t, stream)
	test.RequireGRPCCode(t, err, codes.Unimplemented)
}

func TestCloseReturnsNil(t *testing.T) {
	client := handler.NewClient("http://localhost:9999", handler.NoRetry)
	assert.NoError(t, client.Close())
}
