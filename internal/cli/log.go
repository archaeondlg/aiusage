package cli

import (
	"log/slog"
	"os"
)

// InitLogging configures the global slog logger based on verbose level and format.
func InitLogging(verbose int, jsonFormat bool) {
	var level slog.Level
	switch {
	case verbose >= 3:
		level = slog.LevelDebug
	case verbose >= 2:
		level = slog.LevelInfo
	case verbose >= 1:
		level = slog.LevelWarn
	default:
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if jsonFormat {
		h = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
}
