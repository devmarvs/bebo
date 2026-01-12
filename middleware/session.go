package middleware

import (
	"errors"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/session"
)

const sessionKey = "bebo.session"

type sessionConfig struct {
	clearInvalid bool
}

// SessionOption customizes session middleware behavior.
type SessionOption func(*sessionConfig)

// SessionClearInvalid clears invalid session cookies when enabled.
func SessionClearInvalid(enabled bool) SessionOption {
	return func(cfg *sessionConfig) {
		cfg.clearInvalid = enabled
	}
}

// Session loads a session and stores it on the request context.
func Session(store session.Store, options ...SessionOption) bebo.Middleware {
	cfg := sessionConfig{clearInvalid: true}
	for _, opt := range options {
		opt(&cfg)
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if store == nil {
				return apperr.Internal("session store not configured", nil)
			}

			sess, err := store.Get(ctx.Request)
			if err != nil && !errors.Is(err, session.ErrInvalidCookie) {
				return apperr.Internal("session load failed", err)
			}
			if err != nil && errors.Is(err, session.ErrInvalidCookie) && cfg.clearInvalid {
				store.Clear(ctx.ResponseWriter, sess)
			}

			ctx.Set(sessionKey, sess)
			return next(ctx)
		}
	}
}

// SessionFromContext returns the loaded session.
func SessionFromContext(ctx *bebo.Context) (*session.Session, bool) {
	value, ok := ctx.Get(sessionKey)
	if !ok {
		return nil, false
	}
	sess, ok := value.(*session.Session)
	return sess, ok
}

// SetSession stores a session in context for downstream handlers.
func SetSession(ctx *bebo.Context, sess *session.Session) {
	ctx.Set(sessionKey, sess)
}
