package bebo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestStaticETag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.js")
	if err := os.WriteFile(path, []byte("console.log('ok');"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	app := New()
	app.Static("/static", dir, StaticETag(true))

	req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}

	req = httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", rec.Code)
	}
}

func TestStaticFS(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("ok")},
	}

	app := New()
	app.StaticFS("/static", fsys, StaticETag(false))

	req := httptest.NewRequest(http.MethodGet, "/static/index.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", rec.Body.String())
	}
}

func TestFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	if err := os.WriteFile(path, []byte("home"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	app := New()
	app.File("/", path, StaticETag(false))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "home" {
		t.Fatalf("expected body %q, got %q", "home", rec.Body.String())
	}
}
