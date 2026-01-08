package cache

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/devmarvs/bebo/redis"
)

// RedisOptions configures a Redis cache store.
type RedisOptions struct {
	DisableDefaults bool
	Network         string
	Address         string
	Username        string
	Password        string
	DB              int
	Prefix          string
	DefaultTTL      time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

// RedisStore stores cache entries in Redis.
type RedisStore struct {
	options RedisOptions
	client  *redis.Client
}

// NewRedisStore creates a Redis-backed cache store.
func NewRedisStore(options RedisOptions) *RedisStore {
	cfg := options
	if !cfg.DisableDefaults {
		if cfg.Network == "" {
			cfg.Network = "tcp"
		}
		if cfg.Address == "" {
			cfg.Address = "127.0.0.1:6379"
		}
		if cfg.Prefix == "" {
			cfg.Prefix = "bebo:cache:"
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

	return &RedisStore{options: cfg, client: client}
}

// Get returns a cache entry by key.
func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := s.client.DoContext(ctx, "GET", s.key(key))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	payload, ok := resp.([]byte)
	if !ok {
		return nil, false, errors.New("redis: invalid response")
	}
	return payload, true, nil
}

// Set stores a cache entry with an optional TTL.
func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ttl <= 0 {
		ttl = s.options.DefaultTTL
	}
	args := []string{"SET", s.key(key), string(value)}
	if ttl > 0 {
		ms := ttl.Milliseconds()
		if ms <= 0 {
			ms = 1
		}
		args = append(args, "PX", strconv.FormatInt(ms, 10))
	}
	_, err := s.client.DoContext(ctx, args...)
	return err
}

// Delete removes a cache entry.
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := s.client.DoContext(ctx, "DEL", s.key(key))
	return err
}

func (s *RedisStore) key(key string) string {
	return s.options.Prefix + key
}
