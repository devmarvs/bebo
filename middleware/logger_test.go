package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

type captureHandler struct {
	mu     sync.Mutex
	levels []slog.Level
}

func (c *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (c *captureHandler) Handle(_ context.Context, record slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.levels = append(c.levels, record.Level)
	return nil
}

func (c *captureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return c
}

func (c *captureHandler) WithGroup(string) slog.Handler {
	return c
}

func (c *captureHandler) Levels() []slog.Level {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]slog.Level{}, c.levels...)
}

func TestLoggerOptionsErrorLevel(t *testing.T) {
	handler := &captureHandler{}
	logger := slog.New(handler)

	app := bebo.New(
		bebo.WithLogger(logger),
		bebo.WithErrorHandler(func(*bebo.Context, error) {}),
	)
	app.Use(LoggerWithOptions(LoggerOptions{Fields: []LogField{LogStatus()}, ErrorLevel: true}))
	app.GET("/boom", func(ctx *bebo.Context) error {
		return apperr.Internal("boom", nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	levels := handler.Levels()
	if len(levels) == 0 {
		t.Fatalf("expected log entry")
	}
	if levels[0] != slog.LevelError {
		t.Fatalf("expected error level, got %v", levels[0])
	}
}

func TestLoggerOptionsSkipPath(t *testing.T) {
	handler := &captureHandler{}
	logger := slog.New(handler)

	app := bebo.New(
		bebo.WithLogger(logger),
		bebo.WithErrorHandler(func(*bebo.Context, error) {}),
	)
	app.Use(LoggerWithOptions(LoggerOptions{SkipPaths: []string{"/skip"}}))
	app.GET("/skip", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(handler.Levels()) != 0 {
		t.Fatalf("expected no logs for skipped path")
	}
}
