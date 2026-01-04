package middleware

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter enforces a token-bucket rate limit.
type Limiter struct {
	rate    float64
	burst   float64
	mu      sync.Mutex
	buckets map[string]*bucket
	ttl     time.Duration
}

// LimiterOption customizes the limiter.
type LimiterOption func(*Limiter)

// LimiterTTL evicts idle buckets after the duration.
func LimiterTTL(ttl time.Duration) LimiterOption {
	return func(l *Limiter) {
		l.ttl = ttl
	}
}

// NewLimiter creates a limiter with tokens per second and burst.
func NewLimiter(rate float64, burst int, options ...LimiterOption) *Limiter {
	limiter := &Limiter{
		rate:    rate,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
		ttl:     0,
	}
	for _, opt := range options {
		opt(limiter)
	}
	return limiter
}

// Allow reports whether the key can proceed.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	if l.ttl > 0 && now.Sub(b.last) > l.ttl {
		b.tokens = l.burst
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}

	b.tokens -= 1
	return true
}

// KeyFunc extracts a rate limiting key from the request.
type KeyFunc func(*bebo.Context) string

// LimitHandler handles rate limit violations.
type LimitHandler func(*bebo.Context) error

type rateLimitConfig struct {
	keyFunc    KeyFunc
	onLimit    LimitHandler
	retryAfter time.Duration
}

// RateLimitOption customizes rate limit middleware behavior.
type RateLimitOption func(*rateLimitConfig)

// RateLimitKey sets the key function.
func RateLimitKey(fn KeyFunc) RateLimitOption {
	return func(cfg *rateLimitConfig) {
		cfg.keyFunc = fn
	}
}

// RateLimitHandler sets the handler for limited requests.
func RateLimitHandler(fn LimitHandler) RateLimitOption {
	return func(cfg *rateLimitConfig) {
		cfg.onLimit = fn
	}
}

// RateLimitRetryAfter sets the Retry-After header duration.
func RateLimitRetryAfter(duration time.Duration) RateLimitOption {
	return func(cfg *rateLimitConfig) {
		cfg.retryAfter = duration
	}
}

// RateLimit enforces a token bucket rate limit.
func RateLimit(limiter *Limiter, options ...RateLimitOption) bebo.Middleware {
	cfg := rateLimitConfig{keyFunc: clientIP, retryAfter: 0}
	for _, opt := range options {
		opt(&cfg)
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			if limiter == nil {
				return apperr.Internal("rate limiter not configured", nil)
			}

			key := cfg.keyFunc(ctx)
			if key == "" {
				return next(ctx)
			}

			if !limiter.Allow(key) {
				if cfg.retryAfter > 0 {
					ctx.ResponseWriter.Header().Set("Retry-After", formatRetryAfter(cfg.retryAfter))
				}
				if cfg.onLimit != nil {
					return cfg.onLimit(ctx)
				}
				return apperr.RateLimited("rate limit exceeded", nil)
			}

			return next(ctx)
		}
	}
}

func clientIP(ctx *bebo.Context) string {
	ip := ctx.Request.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(ctx.Request.RemoteAddr)
	if err == nil {
		return host
	}
	return ctx.Request.RemoteAddr
}

func formatRetryAfter(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}
