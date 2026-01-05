package middleware

import "github.com/devmarvs/bebo"

// SecurityHeadersOptions configures security response headers.
type SecurityHeadersOptions struct {
	DisableDefaults           bool
	ContentTypeNosniff        bool
	FrameOptions              string
	ReferrerPolicy            string
	ContentSecurityPolicy     string
	PermissionsPolicy         string
	StrictTransportSecurity   string
	CrossOriginOpenerPolicy   string
	CrossOriginEmbedderPolicy string
	CrossOriginResourcePolicy string
}

// DefaultSecurityHeaders returns the default header settings.
func DefaultSecurityHeaders() SecurityHeadersOptions {
	return SecurityHeadersOptions{
		ContentTypeNosniff: true,
		FrameOptions:       "DENY",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	}
}

// SecurityHeaders adds common security headers.
func SecurityHeaders(options SecurityHeadersOptions) bebo.Middleware {
	if !options.DisableDefaults {
		defaults := DefaultSecurityHeaders()
		if options.FrameOptions == "" {
			options.FrameOptions = defaults.FrameOptions
		}
		if options.ReferrerPolicy == "" {
			options.ReferrerPolicy = defaults.ReferrerPolicy
		}
		if !options.ContentTypeNosniff {
			options.ContentTypeNosniff = defaults.ContentTypeNosniff
		}
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			headers := ctx.ResponseWriter.Header()
			if options.ContentTypeNosniff {
				headers.Set("X-Content-Type-Options", "nosniff")
			}
			if options.FrameOptions != "" {
				headers.Set("X-Frame-Options", options.FrameOptions)
			}
			if options.ReferrerPolicy != "" {
				headers.Set("Referrer-Policy", options.ReferrerPolicy)
			}
			if options.ContentSecurityPolicy != "" {
				headers.Set("Content-Security-Policy", options.ContentSecurityPolicy)
			}
			if options.PermissionsPolicy != "" {
				headers.Set("Permissions-Policy", options.PermissionsPolicy)
			}
			if options.StrictTransportSecurity != "" {
				headers.Set("Strict-Transport-Security", options.StrictTransportSecurity)
			}
			if options.CrossOriginOpenerPolicy != "" {
				headers.Set("Cross-Origin-Opener-Policy", options.CrossOriginOpenerPolicy)
			}
			if options.CrossOriginEmbedderPolicy != "" {
				headers.Set("Cross-Origin-Embedder-Policy", options.CrossOriginEmbedderPolicy)
			}
			if options.CrossOriginResourcePolicy != "" {
				headers.Set("Cross-Origin-Resource-Policy", options.CrossOriginResourcePolicy)
			}
			return next(ctx)
		}
	}
}
