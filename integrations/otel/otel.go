package otel

import (
	"context"

	"github.com/devmarvs/bebo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer adapts OpenTelemetry tracing to middleware.Trace.
type Tracer struct {
	tracer trace.Tracer
}

// NewTracer creates an OpenTelemetry tracer adapter.
func NewTracer(name string) (*Tracer, error) {
	if name == "" {
		name = "bebo"
	}
	return &Tracer{tracer: otel.Tracer(name)}, nil
}

// Start starts an OpenTelemetry span for the request.
func (t *Tracer) Start(ctx *bebo.Context) (context.Context, func(status int, err error)) {
	if t == nil || ctx == nil {
		return context.Background(), nil
	}

	req := ctx.Request
	spanCtx, span := t.tracer.Start(req.Context(), req.Method+" "+req.URL.Path)

	attrs := []attribute.KeyValue{
		attribute.String("http.method", req.Method),
		attribute.String("http.target", req.URL.Path),
		attribute.String("http.scheme", req.URL.Scheme),
	}
	if req.Host != "" {
		attrs = append(attrs, attribute.String("http.host", req.Host))
	}
	if req.UserAgent() != "" {
		attrs = append(attrs, attribute.String("http.user_agent", req.UserAgent()))
	}

	span.SetAttributes(attrs...)

	return spanCtx, func(status int, err error) {
		span.SetAttributes(attribute.Int("http.status_code", status))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}
}
