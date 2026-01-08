package middleware

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/router"
)

// RateLimitPolicy maps a route to an allow function.
type RateLimitPolicy struct {
	Method string
	Host   string
	Path   string
	Allow  AllowFunc
}

// RateLimitPolicies applies rate limits per route.
type RateLimitPolicies struct {
	router   *router.Router
	policies map[router.RouteID]AllowFunc
}

// NewRateLimitPolicies builds a route policy matcher.
func NewRateLimitPolicies(policies []RateLimitPolicy) (*RateLimitPolicies, error) {
	matcher := router.New()
	mapping := make(map[router.RouteID]AllowFunc, len(policies))

	for _, policy := range policies {
		if policy.Path == "" {
			return nil, errors.New("rate limit policy path required")
		}
		if policy.Allow == nil {
			return nil, errors.New("rate limit policy allow function required")
		}
		method := policy.Method
		if method == "" {
			method = "*"
		}
		id, err := matcher.AddWithHost(method, policy.Host, policy.Path)
		if err != nil {
			return nil, err
		}
		mapping[id] = policy.Allow
	}

	return &RateLimitPolicies{router: matcher, policies: mapping}, nil
}

// Middleware returns a middleware that enforces the policies.
func (p *RateLimitPolicies) Middleware(options ...RateLimitOption) bebo.Middleware {
	cfg := rateLimitConfig{keyFunc: clientIP, retryAfter: 0}
	for _, opt := range options {
		opt(&cfg)
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if p == nil || p.router == nil {
				return next(ctx)
			}
			host := requestHost(ctx.Request)
			id, _, ok := p.router.MatchHost(ctx.Request.Method, host, ctx.Request.URL.Path)
			if !ok {
				return next(ctx)
			}
			allow := p.policies[id]
			return applyRateLimit(ctx, allow, cfg, next)
		}
	}
}

func requestHost(r *http.Request) string {
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	if strings.Contains(host, ":") {
		if h, _, err := net.SplitHostPort(host); err == nil {
			return h
		}
	}
	return host
}
