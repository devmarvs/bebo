package bebo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devmarvs/bebo/render"
)

func TestMethodNotAllowed(t *testing.T) {
	app := New()
	app.GET("/users/:id", func(ctx *Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		t.Fatalf("expected Allow header")
	}
}

func TestPath(t *testing.T) {
	app := New()
	app.Route(http.MethodGet, "/users/:id", func(ctx *Context) error {
		return nil
	}, WithName("user.show"))

	path, ok := app.Path("user.show", map[string]string{"id": "42"})
	if !ok {
		t.Fatalf("expected path")
	}
	if path != "/users/42" {
		t.Fatalf("expected /users/42, got %s", path)
	}

	path, ok = app.PathWithQuery("user.show", map[string]string{"id": "42"}, map[string]string{"q": "test"})
	if !ok {
		t.Fatalf("expected path with query")
	}
	if path != "/users/42?q=test" {
		t.Fatalf("expected /users/42?q=test, got %s", path)
	}

	if _, ok := app.Path("user.show", map[string]string{}); ok {
		t.Fatalf("expected missing param to fail")
	}
}

func TestErrorPageHTML(t *testing.T) {
	dir := t.TempDir()
	layout := filepath.Join(dir, "layout.html")
	page := filepath.Join(dir, "error.html")

	if err := os.WriteFile(layout, []byte("{{ template \"content\" . }}"), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if err := os.WriteFile(page, []byte("{{ define \"content\" }}{{ .Message }}{{ end }}"), 0o644); err != nil {
		t.Fatalf("write error page: %v", err)
	}

	engine, err := render.NewEngineWithOptions(dir, render.Options{Layout: "layout.html"})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}

	app := New(
		WithRenderer(engine),
		WithErrorTemplates(map[int]string{http.StatusNotFound: "error.html"}),
	)

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "not found") {
		t.Fatalf("expected error page body")
	}
}
