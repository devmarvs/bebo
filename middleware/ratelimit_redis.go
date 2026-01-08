package middleware

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/devmarvs/bebo/redis"
)

const redisTokenBucketScript = `local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local data = redis.call("HMGET", key, "tokens", "last")
local tokens = tonumber(data[1])
local last = tonumber(data[2])

if tokens == nil then
  tokens = burst
end
if last == nil then
  last = now
end

local delta = math.max(0, now - last)
tokens = math.min(burst, tokens + (delta * rate / 1000))
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

redis.call("HMSET", key, "tokens", tokens, "last", now)
if ttl > 0 then
  redis.call("PEXPIRE", key, ttl)
end
return allowed`

// RedisLimiterOptions configures a Redis-backed limiter.
type RedisLimiterOptions struct {
	DisableDefaults bool
	Network         string
	Address         string
	Username        string
	Password        string
	DB              int
	Prefix          string
	TTL             time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	Now             func() time.Time
}

// RedisLimiter enforces a token bucket rate limit in Redis.
type RedisLimiter struct {
	rate    float64
	burst   float64
	options RedisLimiterOptions
	client  *redis.Client
	now     func() time.Time
}

// NewRedisLimiter creates a Redis-backed limiter.
func NewRedisLimiter(rate float64, burst int, options RedisLimiterOptions) *RedisLimiter {
	cfg := options
	if !cfg.DisableDefaults {
		if cfg.Network == "" {
			cfg.Network = "tcp"
		}
		if cfg.Address == "" {
			cfg.Address = "127.0.0.1:6379"
		}
		if cfg.Prefix == "" {
			cfg.Prefix = "bebo:ratelimit:"
		}
		if cfg.DialTimeout == 0 {
			cfg.DialTimeout = 2 * time.Second
		}
		if cfg.ReadTimeout == 0 {
			cfg.ReadTimeout = 2 * time.Second
		}
		if cfg.WriteTimeout == 0 {
			cfg.WriteTimeout = 2 * time.Second
		}
	}

	client := redis.New(redis.Options{
		Network:      cfg.Network,
		Address:      cfg.Address,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	return &RedisLimiter{
		rate:    rate,
		burst:   float64(burst),
		options: cfg,
		client:  client,
		now:     now,
	}
}

// Allow evaluates a token bucket rate limit for a key.
func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if l == nil {
		return false, errors.New("redis limiter not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if key == "" {
		return true, nil
	}
	now := l.now().UnixNano() / int64(time.Millisecond)
	ttl := l.options.TTL.Milliseconds()

	resp, err := l.client.DoContext(ctx,
		"EVAL",
		redisTokenBucketScript,
		"1",
		l.options.Prefix+key,
		strconv.FormatFloat(l.rate, 'f', -1, 64),
		strconv.FormatFloat(l.burst, 'f', -1, 64),
		strconv.FormatInt(now, 10),
		strconv.FormatInt(ttl, 10),
	)
	if err != nil {
		return false, err
	}

	switch value := resp.(type) {
	case int64:
		return value == 1, nil
	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return false, err
		}
		return parsed == 1, nil
	default:
		return false, errors.New("redis: invalid response")
	}
}

// AllowKey evaluates a token bucket rate limit for a key without a context.
func (l *RedisLimiter) AllowKey(key string) (bool, error) {
	return l.Allow(context.Background(), key)
}
