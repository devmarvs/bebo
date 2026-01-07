package realtime

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/devmarvs/bebo"
)

// SSEOptions configures server-sent events.
type SSEOptions struct {
	Headers map[string]string
}

// SSEMessage is an SSE payload.
type SSEMessage struct {
	Event string
	ID    string
	Data  string
	Retry time.Duration
}

// SSE represents an active SSE stream.
type SSE struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	closed  bool
}

// StartSSE configures response headers and returns an SSE stream.
func StartSSE(ctx *bebo.Context, options SSEOptions) (*SSE, error) {
	flusher, ok := ctx.ResponseWriter.(http.Flusher)
	if !ok {
		return nil, errors.New("response writer does not support flushing")
	}

	headers := ctx.ResponseWriter.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("X-Accel-Buffering", "no")
	for key, value := range options.Headers {
		headers.Set(key, value)
	}

	return &SSE{w: ctx.ResponseWriter, flusher: flusher}, nil
}

// Send writes an SSE message and flushes the response.
func (s *SSE) Send(msg SSEMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("sse stream closed")
	}

	if msg.Retry > 0 {
		fmt.Fprintf(s.w, "retry: %d\n", msg.Retry.Milliseconds())
	}
	if msg.ID != "" {
		fmt.Fprintf(s.w, "id: %s\n", msg.ID)
	}
	if msg.Event != "" {
		fmt.Fprintf(s.w, "event: %s\n", msg.Event)
	}

	for _, line := range strings.Split(msg.Data, "\n") {
		fmt.Fprintf(s.w, "data: %s\n", line)
	}
	fmt.Fprint(s.w, "\n")
	s.flusher.Flush()
	return nil
}

// Close marks the SSE stream as closed.
func (s *SSE) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}
