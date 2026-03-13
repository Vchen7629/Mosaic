package middleware

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"time"
)

// wrapper to extend http response writer to expose
// the status codes
type WrappedWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *WrappedWriter) WriteHeader(statuscode int) {
	w.ResponseWriter.WriteHeader(statuscode)
	w.StatusCode = statuscode
}

// Hijack implements http.Hijacker interface to support WebSocket upgrades
func (w *WrappedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

// logging middleware to track status codes, the url path, and response latency
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &WrappedWriter{
			ResponseWriter: w,
			StatusCode: http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)
		log.Println(wrapped.StatusCode, r.Method, r.URL.Path, time.Since(start))
	})
}