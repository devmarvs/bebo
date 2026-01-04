package bebo

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

	if _, ok := app.Path("user.show", map[string]string{}); ok {
		t.Fatalf("expected missing param to fail")
	}
}
