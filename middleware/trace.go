package middleware

import (
	"context"
	"net/http"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

// Tracer starts spans for incoming requests.
type Tracer interface {
	Start(*bebo.Context) (context.Context, func(status int, err error))
}

// Trace records request spans using the provided tracer.
func Trace(tracer Tracer) bebo.Middleware {
	return TraceWithOptions(TraceOptions{Tracer: tracer})
}

// TraceOptions configures tracing middleware.
type TraceOptions struct {
	Tracer    Tracer
	SkipPaths []string
}

// DefaultTraceOptions returns default tracing options.
func DefaultTraceOptions(tracer Tracer) TraceOptions {
	return TraceOptions{
		Tracer:    tracer,
		SkipPaths: []string{"/metrics", "/health"},
	}
}

// TraceWithOptions records request spans with options.
func TraceWithOptions(options TraceOptions) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if options.Tracer == nil {
				return next(ctx)
			}
			if shouldSkipPath(ctx.Request.URL.Path, options.SkipPaths) {
				return next(ctx)
			}

			recorder := newResponseRecorder(ctx.ResponseWriter)
			ctx.ResponseWriter = recorder

			traceCtx, finish := options.Tracer.Start(ctx)
			if traceCtx != nil {
				ctx.Request = ctx.Request.WithContext(traceCtx)
			}

			err := next(ctx)

			status := recorder.Status()
			if err != nil {
				if appErr := apperr.As(err); appErr != nil {
					status = appErr.Status
				} else if status == 0 {
					status = http.StatusInternalServerError
				}
			}
			if finish != nil {
				finish(status, err)
			}

			return err
		}
	}
}
