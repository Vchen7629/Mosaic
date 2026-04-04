package test

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// grpcDataFrame encodes a proto message as a gRPC-web data frame (type 0x00).
func GrpcDataFrame(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(msg)
	require.NoError(t, err)
	frame := make([]byte, 5+len(b))
	frame[0] = 0x00
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(b)))
	copy(frame[5:], b)
	return frame
}

// grpcTrailerFrame builds a gRPC-web trailer frame (type 0x80).
func GrpcTrailerFrame(code codes.Code, msg string) []byte {
	trailer := fmt.Sprintf("grpc-status: %d", int(code))
	if msg != "" {
		trailer += fmt.Sprintf("\r\ngrpc-message: %s", msg)
	}
	b := []byte(trailer)
	frame := make([]byte, 5+len(b))
	frame[0] = 0x80
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(b)))
	copy(frame[5:], b)
	return frame
}

// serveGRPCWeb spins up a test server that always returns the given raw body with HTTP 200.
func ServeGRPCWeb(body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/grpc-web+proto")
		w.WriteHeader(http.StatusOK)
		w.Write(body) //nolint:errcheck
	}))
}

// serveGRPCWebStatus spins up a test server that returns the given HTTP status with no body.
func ServeGRPCWebStatus(httpStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(httpStatus)
	}))
}

// requireGRPCCode asserts that err is a gRPC status error with the expected code.
func RequireGRPCCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok, "error must be a gRPC status error")
	assert.Equal(t, expected, s.Code())
}
