package middleware

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/devmarvs/bebo"
)

// CacheControl sets a Cache-Control header on responses.
func CacheControl(value string) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if value != "" {
				ctx.ResponseWriter.Header().Set("Cache-Control", value)
			}
			return next(ctx)
		}
	}
}

// ETagOptions configures ETag behavior.
type ETagOptions struct {
	MaxSize int64
	Weak    bool
}

// ETag adds ETag handling for buffered responses.
func ETag(options ETagOptions) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			original := ctx.ResponseWriter
			writer := newETagWriter(original, options.MaxSize)
			ctx.ResponseWriter = writer

			err := next(ctx)
			if err != nil {
				ctx.ResponseWriter = original
				return err
			}

			writer.finalize(ctx.Request, options)
			return nil
		}
	}
}

type etagWriter struct {
	writer   http.ResponseWriter
	header   http.Header
	status   int
	buffer   []byte
	maxSize  int64
	overflow bool
	wrote    bool
}

func newETagWriter(w http.ResponseWriter, maxSize int64) *etagWriter {
	return &etagWriter{writer: w, header: make(http.Header), maxSize: maxSize}
}

func (e *etagWriter) Header() http.Header {
	return e.header
}

func (e *etagWriter) WriteHeader(status int) {
	if e.status == 0 {
		e.status = status
	}
}

func (e *etagWriter) Write(p []byte) (int, error) {
	if e.overflow {
		return e.writer.Write(p)
	}
	if e.maxSize > 0 && int64(len(e.buffer)+len(p)) > e.maxSize {
		e.overflow = true
		e.flushToUnderlying()
		return e.writer.Write(p)
	}
	e.buffer = append(e.buffer, p...)
	return len(p), nil
}

func (e *etagWriter) Flush() {
	if !e.overflow {
		e.overflow = true
		e.flushToUnderlying()
	}
	if flusher, ok := e.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (e *etagWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if !e.overflow {
		e.overflow = true
		e.flushToUnderlying()
	}
	hijacker, ok := e.writer.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (e *etagWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := e.writer.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (e *etagWriter) Unwrap() http.ResponseWriter {
	return e.writer
}

func (e *etagWriter) finalize(r *http.Request, options ETagOptions) {
	if e.overflow || e.wrote {
		return
	}
	status := e.status
	if status == 0 {
		status = http.StatusOK
	}
	if status < 200 || status >= 300 {
		e.flushToUnderlying()
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		e.flushToUnderlying()
		return
	}
	if e.header.Get("ETag") != "" || e.header.Get("Content-Encoding") != "" {
		e.flushToUnderlying()
		return
	}

	hash := sha256.Sum256(e.buffer)
	etag := "\"" + hex.EncodeToString(hash[:]) + "\""
	if options.Weak {
		etag = "W/" + etag
	}
	e.header.Set("ETag", etag)
	if matchETag(r.Header.Get("If-None-Match"), etag) {
		copyHeaders(e.writer.Header(), e.header)
		e.writer.WriteHeader(http.StatusNotModified)
		e.wrote = true
		return
	}

	e.flushToUnderlying()
}

func (e *etagWriter) flushToUnderlying() {
	if e.wrote {
		return
	}
	status := e.status
	if status == 0 {
		status = http.StatusOK
	}
	copyHeaders(e.writer.Header(), e.header)
	e.writer.WriteHeader(status)
	if len(e.buffer) > 0 {
		_, _ = e.writer.Write(e.buffer)
	}
	e.wrote = true
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func matchETag(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, part := range strings.Split(header, ",") {
		if strings.TrimSpace(part) == etag {
			return true
		}
	}
	return false
}
