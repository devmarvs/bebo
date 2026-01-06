package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/metrics"
)

func TestMetricsWithOptionsSkipPath(t *testing.T) {
	registry := metrics.New()
	app := bebo.New()
	app.Use(MetricsWithOptions(MetricsOptions{Registry: registry, SkipPaths: []string{"/skip"}}))

	app.GET("/skip", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})
	app.GET("/ok", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	snap := registry.Snapshot()
	if snap.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", snap.Requests)
	}
}
