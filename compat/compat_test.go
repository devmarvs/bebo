package compat_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/middleware"
	"github.com/devmarvs/bebo/render"
	"github.com/devmarvs/bebo/testutil"
)

func TestNamedRoutesCompatibility(t *testing.T) {
	app := bebo.New()
	app.Route("GET", "/users/:id", func(*bebo.Context) error { return nil }, bebo.WithName("user.show"))

	path, ok := app.Path("user.show", map[string]string{"id": "42"})
	if !ok {
		t.Fatalf("expected named route path")
	}
	if path != "/users/42" {
		t.Fatalf("expected path /users/42, got %q", path)
	}
}

func TestRequestIDCompatibility(t *testing.T) {
	rec, err := testutil.RunMiddleware(t, []bebo.Middleware{middleware.RequestID()}, nil, nil)
	if err != nil {
		t.Fatalf("middleware: %v", err)
	}
	if rec.Header().Get(bebo.RequestIDHeader) == "" {
		t.Fatalf("expected request id header")
	}
}

func TestRenderJSONCompatibility(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := render.JSON(rec, http.StatusCreated, map[string]string{"status": "ok"}); err != nil {
		t.Fatalf("render json: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" {
		t.Fatalf("expected content type")
	}
}
