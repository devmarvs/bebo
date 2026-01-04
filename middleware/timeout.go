package middleware

import (
	"time"

	"github.com/devmarvs/bebo"
)

// Timeout enforces a request timeout.
func Timeout(duration time.Duration) bebo.Middleware {
	return func(next bebo.Handler) bebo.Handler {
		return bebo.TimeoutHandler(next, duration)
	}
}
