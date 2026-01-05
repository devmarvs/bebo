package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
)

func TestIPFilterAllow(t *testing.T) {
	filter, err := IPFilter(IPFilterOptions{Allow: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatalf("ip filter: %v", err)
	}

	app := bebo.New()
	app.Use(filter)
	app.GET("/", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestIPFilterDeny(t *testing.T) {
	filter, err := IPFilter(IPFilterOptions{Allow: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatalf("ip filter: %v", err)
	}

	app := bebo.New()
	app.Use(filter)
	app.GET("/", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
