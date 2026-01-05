package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
)

func TestSecurityHeaders(t *testing.T) {
	app := bebo.New()
	app.Use(SecurityHeaders(DefaultSecurityHeaders()))

	app.GET("/", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff header")
	}
	if rec.Header().Get("X-Frame-Options") == "" {
		t.Fatalf("expected frame options header")
	}
	if rec.Header().Get("Referrer-Policy") == "" {
		t.Fatalf("expected referrer policy header")
	}
}
