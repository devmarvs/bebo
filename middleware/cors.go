package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/devmarvs/bebo"
)

// CORSOptions configures CORS behavior.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// CORS enables cross-origin requests.
func CORS(options CORSOptions) bebo.Middleware {
	opts := normalizeCORS(options)
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			origin := ctx.Request.Header.Get("Origin")
			if origin != "" {
				if allowedOrigin, ok := matchOrigin(opts.AllowedOrigins, origin, opts.AllowCredentials); ok {
					ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					ctx.ResponseWriter.Header().Add("Vary", "Origin")
					if opts.AllowCredentials {
						ctx.ResponseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
					}
					if len(opts.ExposedHeaders) > 0 {
						ctx.ResponseWriter.Header().Set("Access-Control-Expose-Headers", strings.Join(opts.ExposedHeaders, ", "))
					}
				}
			}

			if ctx.Request.Method == http.MethodOptions && ctx.Request.Header.Get("Access-Control-Request-Method") != "" {
				ctx.ResponseWriter.Header().Set("Access-Control-Allow-Methods", strings.Join(opts.AllowedMethods, ", "))
				ctx.ResponseWriter.Header().Set("Access-Control-Allow-Headers", strings.Join(opts.AllowedHeaders, ", "))
				if opts.MaxAge > 0 {
					ctx.ResponseWriter.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(opts.MaxAge.Seconds())))
				}
				ctx.ResponseWriter.WriteHeader(http.StatusNoContent)
				return nil
			}

			return next(ctx)
		}
	}
}

func normalizeCORS(options CORSOptions) CORSOptions {
	if len(options.AllowedOrigins) == 0 {
		options.AllowedOrigins = []string{"*"}
	}
	if len(options.AllowedMethods) == 0 {
		options.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(options.AllowedHeaders) == 0 {
		options.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	return options
}

func matchOrigin(allowed []string, origin string, allowCredentials bool) (string, bool) {
	for _, entry := range allowed {
		if entry == "*" {
			if allowCredentials {
				return origin, true
			}
			return "*", true
		}
		if strings.EqualFold(entry, origin) {
			return origin, true
		}
	}
	return "", false
}
