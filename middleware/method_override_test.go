package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devmarvs/bebo"
)

func TestMethodOverrideFromForm(t *testing.T) {
	app := bebo.New()
	app.UsePre(MethodOverride(MethodOverrideOptions{}))
	app.DELETE("/items/:id", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/items/123", strings.NewReader("_method=DELETE"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMethodOverrideRejectsInvalid(t *testing.T) {
	app := bebo.New()
	app.UsePre(MethodOverride(MethodOverrideOptions{}))
	app.PUT("/items/:id", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/items/123", strings.NewReader("_method=TRACE"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
