package middleware

import (
	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

type authConfig struct {
	unauthorizedMessage string
	forbiddenMessage    string
}

// AuthOption customizes auth middleware behavior.
type AuthOption func(*authConfig)

// AuthUnauthorizedMessage sets the unauthorized message.
func AuthUnauthorizedMessage(message string) AuthOption {
	return func(cfg *authConfig) {
		cfg.unauthorizedMessage = message
	}
}

// AuthForbiddenMessage sets the forbidden message.
func AuthForbiddenMessage(message string) AuthOption {
	return func(cfg *authConfig) {
		cfg.forbiddenMessage = message
	}
}

// RequireAuth ensures a request is authenticated.
func RequireAuth(auth bebo.Authenticator, options ...AuthOption) bebo.Middleware {
	return RequireAuthorization(auth, nil, options...)
}

// RequireAuthorization ensures a request is authenticated and authorized.
func RequireAuthorization(auth bebo.Authenticator, authorizer bebo.Authorizer, options ...AuthOption) bebo.Middleware {
	cfg := authConfig{unauthorizedMessage: "unauthorized", forbiddenMessage: "forbidden"}
	for _, opt := range options {
		opt(&cfg)
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if auth == nil {
				return apperr.Internal("authenticator not configured", nil)
			}

			principal, err := auth.Authenticate(ctx)
			if err != nil || principal == nil {
				return apperr.Unauthorized(cfg.unauthorizedMessage, err)
			}

			bebo.SetPrincipal(ctx, principal)

			if authorizer != nil {
				if err := authorizer.Authorize(ctx, principal); err != nil {
					return apperr.Forbidden(cfg.forbiddenMessage, err)
				}
			}

			return next(ctx)
		}
	}
}
