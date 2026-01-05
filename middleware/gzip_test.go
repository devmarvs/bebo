package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
)

func TestGzipMiddleware(t *testing.T) {
	app := bebo.New()
	app.Use(Gzip(0))

	app.GET("/", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content encoding")
	}

	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("expected body hello, got %s", string(body))
	}
}
