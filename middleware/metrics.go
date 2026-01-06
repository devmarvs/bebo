package middleware

import (
	"net/http"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/metrics"
)

// Metrics records request metrics into the registry.
func Metrics(registry *metrics.Registry) bebo.Middleware {
	return MetricsWithOptions(MetricsOptions{Registry: registry})
}

// MetricsOptions configures metrics recording.
type MetricsOptions struct {
	Registry  *metrics.Registry
	SkipPaths []string
}

// DefaultMetricsOptions returns default metrics options.
func DefaultMetricsOptions(registry *metrics.Registry) MetricsOptions {
	return MetricsOptions{
		Registry:  registry,
		SkipPaths: []string{"/metrics", "/health"},
	}
}

// MetricsWithOptions records request metrics with options.
func MetricsWithOptions(options MetricsOptions) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if options.Registry == nil {
				return next(ctx)
			}
			if shouldSkipPath(ctx.Request.URL.Path, options.SkipPaths) {
				return next(ctx)
			}

			start := options.Registry.Start()
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

			options.Registry.End(start, status, err)
			return err
		}
	}
}
