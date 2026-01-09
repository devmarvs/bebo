package bebo

import (
	"context"
	"log/slog"
	"net/http"
)

// Logger wraps slog.Logger with request context.
type Logger struct {
	logger    *slog.Logger
	requestID string
	traceID   string
	spanID    string
}

// Info logs an info message.
func (l Logger) Info(msg string, attrs ...slog.Attr) {
	l.logger.Info(msg, l.appendContextFields(attrs)...)
}

// Warn logs a warning message.
func (l Logger) Warn(msg string, attrs ...slog.Attr) {
	l.logger.Warn(msg, l.appendContextFields(attrs)...)
}

// Error logs an error message.
func (l Logger) Error(msg string, attrs ...slog.Attr) {
	l.logger.Error(msg, l.appendContextFields(attrs)...)
}

// Debug logs a debug message.
func (l Logger) Debug(msg string, attrs ...slog.Attr) {
	l.logger.Debug(msg, l.appendContextFields(attrs)...)
}

func (l Logger) appendContextFields(attrs []slog.Attr) []any {
	additional := 0
	if l.requestID != "" {
		additional++
	}
	if l.traceID != "" {
		additional++
	}
	if l.spanID != "" {
		additional++
	}

	out := make([]any, 0, len(attrs)+additional)
	for _, attr := range attrs {
		out = append(out, attr)
	}
	if l.requestID != "" {
		out = append(out, slog.String("request_id", l.requestID))
	}
	if l.traceID != "" {
		out = append(out, slog.String("trace_id", l.traceID))
	}
	if l.spanID != "" {
		out = append(out, slog.String("span_id", l.spanID))
	}
	return out
}

// LoggerFromContext builds a logger that includes request metadata.
func LoggerFromContext(ctx context.Context, logger *slog.Logger) Logger {
	metadata := RequestMetadataFromContext(ctx)
	traceID, spanID, _ := TraceIDs(metadata.Traceparent)
	return Logger{logger: logger, requestID: metadata.RequestID, traceID: traceID, spanID: spanID}
}

// LoggerFromRequest builds a logger using request metadata.
func LoggerFromRequest(r *http.Request, logger *slog.Logger) Logger {
	metadata := RequestMetadataFromRequest(r)
	traceID, spanID, _ := TraceIDs(metadata.Traceparent)
	return Logger{logger: logger, requestID: metadata.RequestID, traceID: traceID, spanID: spanID}
}
