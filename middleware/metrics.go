package middleware

import (
	"net/http"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/metrics"
)

// Metrics records request metrics into the registry.
func Metrics(registry *metrics.Registry) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if registry == nil {
				return next(ctx)
			}

			start := registry.Start()
			recorder := newResponseRecorder(ctx.ResponseWriter)
			ctx.ResponseWriter = recorder

			err := next(ctx)

			status := recorder.Status()
			if err != nil {
				if appErr := apperr.As(err); appErr != nil {
					status = appErr.Status
				} else if status == 0 {
					status = http.StatusInternalServerError
				}
			}

			registry.End(start, status, err)
			return err
		}
	}
}
