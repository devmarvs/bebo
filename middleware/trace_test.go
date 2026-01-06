package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
)

type testTracer struct {
	starts int
}

func (t *testTracer) Start(ctx *bebo.Context) (context.Context, func(status int, err error)) {
	t.starts++
	return ctx.Request.Context(), func(int, error) {}
}

func TestTraceWithOptionsSkipPath(t *testing.T) {
	tracer := &testTracer{}
	app := bebo.New()
	app.Use(TraceWithOptions(TraceOptions{Tracer: tracer, SkipPaths: []string{"/skip"}}))

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

	if tracer.starts != 1 {
		t.Fatalf("expected 1 trace start, got %d", tracer.starts)
	}
}
