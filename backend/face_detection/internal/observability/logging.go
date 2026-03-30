package observability

import (
	"log/slog"
	"os"
)

// returns the structured logger with appropriate log level based on prodMode
func StructuredLogger(prodMode bool) *slog.Logger {
	level := slog.LevelDebug
	if prodMode {
		level = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})

	return slog.New(h).With("service", "face_detection")
}
