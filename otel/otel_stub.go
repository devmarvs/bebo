//go:build !otel

package otel

import (
	"context"
	"errors"

	"github.com/devmarvs/bebo"
)

// ErrUnavailable indicates OpenTelemetry support is disabled.
var ErrUnavailable = errors.New("otel build tag not enabled")

// Tracer is a no-op tracer when OpenTelemetry is disabled.
type Tracer struct{}

// NewTracer returns ErrUnavailable when the otel build tag is not enabled.
func NewTracer(name string) (*Tracer, error) {
	_ = name
	return nil, ErrUnavailable
}

// Start implements the middleware tracer interface as a no-op.
func (t *Tracer) Start(ctx *bebo.Context) (context.Context, func(status int, err error)) {
	if ctx == nil || ctx.Request == nil {
		return context.Background(), func(int, error) {}
	}
	return ctx.Request.Context(), func(int, error) {}
}
