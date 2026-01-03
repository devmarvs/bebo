package bebo

import "log/slog"

// Logger wraps slog.Logger with request context.
type Logger struct {
	logger    *slog.Logger
	requestID string
}

// Info logs an info message.
func (l Logger) Info(msg string, attrs ...slog.Attr) {
	l.logger.Info(msg, l.appendRequestID(attrs)...)
}

// Warn logs a warning message.
func (l Logger) Warn(msg string, attrs ...slog.Attr) {
	l.logger.Warn(msg, l.appendRequestID(attrs)...)
}

// Error logs an error message.
func (l Logger) Error(msg string, attrs ...slog.Attr) {
	l.logger.Error(msg, l.appendRequestID(attrs)...)
}

// Debug logs a debug message.
func (l Logger) Debug(msg string, attrs ...slog.Attr) {
	l.logger.Debug(msg, l.appendRequestID(attrs)...)
}

func (l Logger) appendRequestID(attrs []slog.Attr) []any {
	if l.requestID == "" {
		out := make([]any, 0, len(attrs))
		for _, attr := range attrs {
			out = append(out, attr)
		}
		return out
	}

	out := make([]any, 0, len(attrs)+1)
	for _, attr := range attrs {
		out = append(out, attr)
	}
	out = append(out, slog.String("request_id", l.requestID))
	return out
}
