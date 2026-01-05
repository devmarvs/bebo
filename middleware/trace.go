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
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if tracer == nil {
				return next(ctx)
			}

			recorder := newResponseRecorder(ctx.ResponseWriter)
			ctx.ResponseWriter = recorder

			traceCtx, finish := tracer.Start(ctx)
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
