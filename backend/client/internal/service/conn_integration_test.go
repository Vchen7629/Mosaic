//go:build integration

package service_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"mosaic-client.com/internal/service"
	"mosaic-client.com/internal/test"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// drainHandler upgrades the connection and discards all incoming messages
// so WriteJSON calls on the client side don't block.
func drainHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := testUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// TestSafeConn_WriteJSON_Concurrent verifies that concurrent WriteJSON calls
// do not produce a data race. Run with -race to catch mutex removal regressions.
func TestSafeConn_WriteJSON_Concurrent(t *testing.T) {
	rawConn, cleanup := test.DialWS(t, http.HandlerFunc(drainHandler))
	defer cleanup()

	safe := &service.SafeConn{Conn: rawConn}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			err := safe.WriteJSON(map[string]string{"msg": "hello"})
			require.NoError(t, err)
		}()
	}

	wg.Wait()
}
