package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var captureUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSCapture spins up a test WebSocket server that records every JSON message
// sent to it. Use Conn to create a SafeConn in the test, and Messages() to
// inspect what was written.
type WSCapture struct {
	Conn *websocket.Conn

	mu   sync.Mutex
	msgs []json.RawMessage
	srv  *httptest.Server
}

// NewWSCapture creates a capture server and dials it, returning a WSCapture
// whose Conn field is ready to wrap in a SafeConn. The server and connection
// are closed automatically via t.Cleanup.
func NewWSCapture(t *testing.T) *WSCapture {
	t.Helper()
	c := &WSCapture{}

	c.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverConn, err := captureUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer serverConn.Close()
		for {
			_, msg, err := serverConn.ReadMessage()
			if err != nil {
				break
			}
			c.mu.Lock()
			c.msgs = append(c.msgs, json.RawMessage(msg))
			c.mu.Unlock()
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(c.srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	c.Conn = conn

	t.Cleanup(func() {
		conn.Close()
		c.srv.Close()
	})

	return c
}

// Messages returns a snapshot of all JSON messages received so far.
func (c *WSCapture) Messages() []json.RawMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]json.RawMessage{}, c.msgs...)
}
