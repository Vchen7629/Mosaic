package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// spins up a ws socket
func DialWS(t *testing.T, handler http.Handler) (*websocket.Conn, func()) {
	server := httptest.NewServer(handler)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn, server.Close
}