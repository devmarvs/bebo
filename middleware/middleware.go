package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

// RequestID ensures a request id header is present.
func RequestID() bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			requestID := ctx.RequestID()
			if requestID == "" {
				requestID = bebo.NewRequestID()
				if requestID != "" {
					ctx.Request.Header.Set(bebo.RequestIDHeader, requestID)
				}
			}
			if requestID != "" {
				ctx.ResponseWriter.Header().Set(bebo.RequestIDHeader, requestID)
			}
			return next(ctx)
		}
	}
}

// Recover converts panics into internal errors.
func Recover() bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					err = apperr.Internal("panic", fmt.Errorf("%v", rec))
				}
			}()
			return next(ctx)
		}
	}
}

// Logger logs request/response details.
func Logger() bebo.Middleware {
	return LoggerWith(DefaultLogFields()...)
}

// LoggerWith logs request/response details using custom fields.
func LoggerWith(fields ...LogField) bebo.Middleware {
	if len(fields) == 0 {
		fields = DefaultLogFields()
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			start := time.Now()
			recorder := newResponseRecorder(ctx.ResponseWriter)
			ctx.ResponseWriter = recorder

			err := next(ctx)

			status := recorder.Status()
			if err != nil {
				if appErr := apperr.As(err); appErr != nil {
					status = appErr.Status
				} else if status == 0 {
					status = http.StatusInternalServerError
				}
			}
			if recorder.status == 0 {
				recorder.status = status
			}

			duration := time.Since(start)
			attrs := make([]slog.Attr, 0, len(fields))
			for _, field := range fields {
				attrs = append(attrs, field(ctx, recorder, duration))
			}

			ctx.Logger().Info("request completed", attrs...)
			return err
		}
	}
}

// responseRecorder captures status and response size.
type responseRecorder struct {
	writer http.ResponseWriter
	status int
	bytes  int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{writer: w}
}

func (r *responseRecorder) Header() http.Header {
	return r.writer.Header()
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.writer.WriteHeader(status)
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.writer.Write(p)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *responseRecorder) Bytes() int {
	return r.bytes
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (r *responseRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.writer.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.writer
}
