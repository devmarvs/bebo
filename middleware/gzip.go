package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/devmarvs/bebo"
)

// Gzip compresses responses when supported by the client.
func Gzip(level int) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if ctx.Request.Method == http.MethodHead {
				return next(ctx)
			}
			if !acceptsGzip(ctx.Request) {
				return next(ctx)
			}
			if ctx.ResponseWriter.Header().Get("Content-Encoding") != "" {
				return next(ctx)
			}

			writer := newGzipWriter(ctx.ResponseWriter, level)
			if writer == nil {
				return next(ctx)
			}
			ctx.ResponseWriter = writer

			err := next(ctx)
			_ = writer.Close()
			return err
		}
	}
}

type gzipWriter struct {
	writer   http.ResponseWriter
	gzipper  *gzip.Writer
	status   int
	disabled bool
}

func newGzipWriter(w http.ResponseWriter, level int) *gzipWriter {
	if level == 0 {
		level = gzip.DefaultCompression
	}
	gz, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil
	}

	return &gzipWriter{writer: w, gzipper: gz}
}

func (g *gzipWriter) Header() http.Header {
	return g.writer.Header()
}

func (g *gzipWriter) WriteHeader(status int) {
	g.status = status
	if status == http.StatusNoContent || status == http.StatusNotModified {
		g.disabled = true
		g.writer.WriteHeader(status)
		return
	}
	g.writer.Header().Set("Content-Encoding", "gzip")
	g.writer.Header().Add("Vary", "Accept-Encoding")
	g.writer.Header().Del("Content-Length")
	g.writer.WriteHeader(status)
}

func (g *gzipWriter) Write(p []byte) (int, error) {
	if g.status == 0 {
		g.WriteHeader(http.StatusOK)
	}
	if g.disabled {
		return g.writer.Write(p)
	}
	return g.gzipper.Write(p)
}

func (g *gzipWriter) Close() error {
	if g.disabled || g.gzipper == nil {
		return nil
	}
	return g.gzipper.Close()
}

func (g *gzipWriter) Flush() {
	if flusher, ok := g.writer.(http.Flusher); ok {
		if g.gzipper != nil {
			_ = g.gzipper.Flush()
		}
		flusher.Flush()
	}
}

func acceptsGzip(r *http.Request) bool {
	encoding := r.Header.Get("Accept-Encoding")
	return strings.Contains(encoding, "gzip")
}
