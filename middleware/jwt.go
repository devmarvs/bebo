package middleware

import (
	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/auth"
)

// JWTOptions configures JWT auth middleware.
type JWTOptions struct {
	Authenticator       auth.JWTAuthenticator
	Authorizer          bebo.Authorizer
	UnauthorizedMessage string
	ForbiddenMessage    string
}

// JWT validates bearer tokens using the JWT authenticator.
func JWT(options JWTOptions) bebo.Middleware {
	opts := make([]AuthOption, 0, 2)
	if options.UnauthorizedMessage != "" {
		opts = append(opts, AuthUnauthorizedMessage(options.UnauthorizedMessage))
	}
	if options.ForbiddenMessage != "" {
		opts = append(opts, AuthForbiddenMessage(options.ForbiddenMessage))
	}
	return RequireAuthorization(options.Authenticator, options.Authorizer, opts...)
}

// RequireJWT validates bearer tokens using the JWT authenticator.
func RequireJWT(authenticator auth.JWTAuthenticator, options ...AuthOption) bebo.Middleware {
	return RequireAuth(authenticator, options...)
}
