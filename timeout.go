package bebo

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/devmarvs/bebo/apperr"
)

// TimeoutHandler wraps a handler with a timeout.
func TimeoutHandler(next Handler, duration time.Duration) Handler {
	return func(ctx *Context) error {
		if duration <= 0 {
			return next(ctx)
		}

		reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), duration)
		defer cancel()

		writer := newTimeoutWriter(ctx.ResponseWriter)
		ctx.ResponseWriter = writer
		ctx.Request = ctx.Request.WithContext(reqCtx)

		done := make(chan error, 1)
		go func() {
			done <- next(ctx)
		}()

		select {
		case err := <-done:
			writer.commit()
			return err
		case <-reqCtx.Done():
			writer.timeout()
			return apperr.Timeout("request timeout", reqCtx.Err())
		}
	}
}

type timeoutWriter struct {
	w        http.ResponseWriter
	header   http.Header
	buffer   bytes.Buffer
	code     int
	timedOut bool
	mu       sync.Mutex
}

func newTimeoutWriter(w http.ResponseWriter) *timeoutWriter {
	return &timeoutWriter{w: w, header: make(http.Header)}
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.header
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	if tw.code == 0 {
		tw.code = code
	}
	tw.mu.Unlock()
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	if tw.timedOut {
		tw.mu.Unlock()
		return 0, http.ErrHandlerTimeout
	}
	tw.mu.Unlock()
	return tw.buffer.Write(p)
}

func (tw *timeoutWriter) commit() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return
	}
	for key, values := range tw.header {
		for _, value := range values {
			tw.w.Header().Add(key, value)
		}
	}
	if tw.code != 0 {
		tw.w.WriteHeader(tw.code)
	}
	_, _ = tw.w.Write(tw.buffer.Bytes())
}

func (tw *timeoutWriter) timeout() {
	tw.mu.Lock()
	tw.timedOut = true
	tw.mu.Unlock()
}
