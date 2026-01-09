package middleware

import (
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/devmarvs/bebo"
)

// LogField builds a structured log attribute.
type LogField func(*bebo.Context, *responseRecorder, time.Duration) slog.Attr

// DefaultLogFields returns the standard access log fields.
func DefaultLogFields() []LogField {
	return []LogField{
		LogMethod(),
		LogPath(),
		LogRequestID(),
		LogTraceID(),
		LogSpanID(),
		LogStatus(),
		LogDuration(),
		LogRequestBytes(),
		LogBytes(),
	}
}

// LogMethod logs the HTTP method.
func LogMethod() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		return slog.String("method", ctx.Request.Method)
	}
}

// LogPath logs the request path.
func LogPath() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		return slog.String("path", ctx.Request.URL.Path)
	}
}

// LogStatus logs the response status.
func LogStatus() LogField {
	return func(_ *bebo.Context, recorder *responseRecorder, _ time.Duration) slog.Attr {
		return slog.Int("status", recorder.Status())
	}
}

// LogDuration logs request latency.
func LogDuration() LogField {
	return func(_ *bebo.Context, _ *responseRecorder, duration time.Duration) slog.Attr {
		return slog.Duration("duration", duration)
	}
}

// LogBytes logs response size in bytes.
func LogBytes() LogField {
	return func(_ *bebo.Context, recorder *responseRecorder, _ time.Duration) slog.Attr {
		return slog.Int("bytes", recorder.Bytes())
	}
}

// LogRequestBytes logs request size in bytes (from Content-Length when provided).
func LogRequestBytes() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		return slog.Int64("request_bytes", ctx.Request.ContentLength)
	}
}

// LogTraceID logs the trace id from the traceparent header.
func LogTraceID() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		traceID, _, ok := bebo.TraceIDs(ctx.Request.Header.Get(bebo.TraceparentHeader))
		if !ok {
			return slog.String("trace_id", "")
		}
		return slog.String("trace_id", traceID)
	}
}

// LogSpanID logs the span id from the traceparent header.
func LogSpanID() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		_, spanID, ok := bebo.TraceIDs(ctx.Request.Header.Get(bebo.TraceparentHeader))
		if !ok {
			return slog.String("span_id", "")
		}
		return slog.String("span_id", spanID)
	}
}

// LogRemoteAddr logs the client IP.
func LogRemoteAddr() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		host, _, err := net.SplitHostPort(ctx.Request.RemoteAddr)
		if err == nil {
			return slog.String("remote_addr", host)
		}
		return slog.String("remote_addr", ctx.Request.RemoteAddr)
	}
}

// LogUserAgent logs the user agent.
func LogUserAgent() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		return slog.String("user_agent", ctx.Request.UserAgent())
	}
}

// LogQuery logs the raw query string.
func LogQuery() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		return slog.String("query", strings.TrimPrefix(ctx.Request.URL.RawQuery, "?"))
	}
}

// LogRequestID logs the request id.
func LogRequestID() LogField {
	return func(ctx *bebo.Context, _ *responseRecorder, _ time.Duration) slog.Attr {
		return slog.String("request_id", ctx.RequestID())
	}
}
