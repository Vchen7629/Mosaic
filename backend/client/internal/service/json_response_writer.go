package service

import (
	"sync"

	"github.com/gorilla/websocket"
)

type SafeConn struct {
	Mu   sync.Mutex
	Conn *websocket.Conn
}

// thread safe write json to prevent issues with
// multiple go routines writing
func (s *SafeConn) WriteJSON(v any) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.Conn.WriteJSON(v)
}