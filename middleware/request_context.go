package middleware

import "github.com/devmarvs/bebo"

// RequestContext stores request metadata in the request context.
func RequestContext() bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			metadata := bebo.RequestMetadataFromRequest(ctx.Request)
			ctx.Request = ctx.Request.WithContext(bebo.WithRequestMetadata(ctx.Request.Context(), metadata))
			return next(ctx)
		}
	}
}
