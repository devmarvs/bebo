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
	rate        float64
	burst       float64
	mu          sync.Mutex
	buckets     map[string]*bucket
	ttl         time.Duration
	lastCleanup time.Time
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

	if l.ttl > 0 && (l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.ttl) {
		l.cleanup(now)
		l.lastCleanup = now
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	if l.ttl > 0 && now.Sub(b.last) >= l.ttl {
		delete(l.buckets, key)
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
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

func (l *Limiter) cleanup(now time.Time) {
	if l.ttl <= 0 {
		return
	}
	for key, entry := range l.buckets {
		if now.Sub(entry.last) >= l.ttl {
			delete(l.buckets, key)
		}
	}
}

// AllowFunc evaluates whether a request should proceed.
type AllowFunc func(*bebo.Context, string) (bool, error)

// KeyFunc extracts a rate limiting key from the request.
type KeyFunc func(*bebo.Context) string

// LimitHandler handles rate limit violations.
type LimitHandler func(*bebo.Context) error

// ErrorHandler handles rate limiter errors.
type ErrorHandler func(*bebo.Context, error) error

type rateLimitConfig struct {
	keyFunc    KeyFunc
	onLimit    LimitHandler
	onError    ErrorHandler
	retryAfter time.Duration
	failOpen   bool
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

// RateLimitOnError sets the handler for limiter errors.
func RateLimitOnError(fn ErrorHandler) RateLimitOption {
	return func(cfg *rateLimitConfig) {
		cfg.onError = fn
	}
}

// RateLimitFailOpen allows requests when the limiter errors.
func RateLimitFailOpen(enabled bool) RateLimitOption {
	return func(cfg *rateLimitConfig) {
		cfg.failOpen = enabled
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
	return RateLimitWith(func(_ *bebo.Context, key string) (bool, error) {
		if limiter == nil {
			return false, apperr.Internal("rate limiter not configured", nil)
		}
		return limiter.Allow(key), nil
	}, options...)
}

// RateLimitWith enforces a rate limit using a custom allow function.
func RateLimitWith(allow AllowFunc, options ...RateLimitOption) bebo.Middleware {
	cfg := rateLimitConfig{keyFunc: clientIPKey, retryAfter: 0}
	for _, opt := range options {
		opt(&cfg)
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			return applyRateLimit(ctx, allow, cfg, next)
		}
	}
}

func applyRateLimit(ctx *bebo.Context, allow AllowFunc, cfg rateLimitConfig, next bebo.Handler) error {
	if allow == nil {
		return apperr.Internal("rate limiter not configured", nil)
	}
	if next == nil {
		next = func(*bebo.Context) error { return nil }
	}

	key := cfg.keyFunc(ctx)
	if key == "" {
		return next(ctx)
	}

	allowed, err := allow(ctx, key)
	if err != nil {
		if cfg.onError != nil {
			return cfg.onError(ctx, err)
		}
		if cfg.failOpen {
			return next(ctx)
		}
		if appErr := apperr.As(err); appErr != nil {
			return appErr
		}
		return apperr.Internal("rate limiter error", err)
	}

	if !allowed {
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

func clientIPKey(ctx *bebo.Context) string {
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
