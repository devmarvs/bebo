package middleware

import (
	"net/http"

	"github.com/devmarvs/bebo"
)

// BodyLimit caps the request body size.
func BodyLimit(maxBytes int64) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			ctx.Request.Body = http.MaxBytesReader(ctx.ResponseWriter, ctx.Request.Body, maxBytes)
			return next(ctx)
		}
	}
}
