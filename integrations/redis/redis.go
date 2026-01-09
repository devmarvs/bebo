package redis

import (
	"github.com/devmarvs/bebo/cache"
	"github.com/devmarvs/bebo/middleware"
	"github.com/devmarvs/bebo/redis"
	"github.com/devmarvs/bebo/session"
)

type Client = redis.Client
type Options = redis.Options

func New(options Options) *Client {
	return redis.New(options)
}

type CacheStore = cache.RedisStore
type CacheOptions = cache.RedisOptions

func NewCacheStore(options CacheOptions) *CacheStore {
	return cache.NewRedisStore(options)
}

type SessionStore = session.RedisStore
type SessionOptions = session.RedisOptions

func NewSessionStore(options SessionOptions) *SessionStore {
	return session.NewRedisStore(options)
}

type Limiter = middleware.RedisLimiter
type LimiterOptions = middleware.RedisLimiterOptions

func NewLimiter(rate float64, burst int, options LimiterOptions) *Limiter {
	return middleware.NewRedisLimiter(rate, burst, options)
}
