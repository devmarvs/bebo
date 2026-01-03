package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Options configures logging behavior.
type Options struct {
	Level  string
	Format string
}

// NewLogger builds a slog.Logger with sane defaults.
func NewLogger(options Options) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(options.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handlerOptions := &slog.HandlerOptions{Level: level}
	format := strings.ToLower(options.Format)
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, handlerOptions))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, handlerOptions))
}
