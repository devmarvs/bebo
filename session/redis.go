package session

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/devmarvs/bebo/redis"
)

// RedisOptions configures a Redis store.
type RedisOptions struct {
	DisableDefaults bool
	Name            string
	Network         string
	Address         string
	Username        string
	Password        string
	DB              int
	TTL             time.Duration
	Prefix          string
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	Path            string
	Secure          bool
	HTTPOnly        bool
	SameSite        http.SameSite
}

// RedisStore stores sessions in Redis using a session ID cookie.
type RedisStore struct {
	options RedisOptions
	client  *redis.Client
}

// NewRedisStore creates a Redis-backed session store.
func NewRedisStore(options RedisOptions) *RedisStore {
	cfg := options
	if !cfg.DisableDefaults {
		if cfg.Name == "" {
			cfg.Name = "bebo_session"
		}
		if cfg.Network == "" {
			cfg.Network = "tcp"
		}
		if cfg.Address == "" {
			cfg.Address = "127.0.0.1:6379"
		}
		if cfg.Prefix == "" {
			cfg.Prefix = "bebo:sessions:"
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
		if cfg.Path == "" {
			cfg.Path = "/"
		}
		if !cfg.HTTPOnly {
			cfg.HTTPOnly = true
		}
		if cfg.SameSite == 0 {
			cfg.SameSite = http.SameSiteLaxMode
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

// Get loads a session from the request.
func (s *RedisStore) Get(r *http.Request) (*Session, error) {
	values := map[string]string{}
	cookie, err := r.Cookie(s.options.Name)
	if err != nil || cookie.Value == "" {
		id, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: id, Values: values, store: s, isNew: true}, nil
	}

	payload, err := s.get(cookie.Value)
	if errors.Is(err, redis.ErrNil) {
		id, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: id, Values: values, store: s, isNew: true}, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return &Session{ID: cookie.Value, Values: values, store: s}, nil
}

// Save persists a session.
func (s *RedisStore) Save(w http.ResponseWriter, session *Session) error {
	if session == nil {
		return errors.New("session missing")
	}
	id := session.ID
	if id == "" {
		newID, err := newSessionID()
		if err != nil {
			return err
		}
		id = newID
		session.ID = id
	}

	payload, err := json.Marshal(session.Values)
	if err != nil {
		return err
	}

	if err := s.set(id, payload); err != nil {
		return err
	}

	s.setCookie(w, id)
	session.isNew = false
	return nil
}

// Clear removes a session.
func (s *RedisStore) Clear(w http.ResponseWriter, session *Session) {
	if session != nil && session.ID != "" {
		_ = s.del(session.ID)
	}
	s.clearCookie(w)
	if session != nil {
		session.Values = map[string]string{}
		session.isNew = true
		session.ID = ""
	}
}

func (s *RedisStore) setCookie(w http.ResponseWriter, id string) {
	cookie := &http.Cookie{
		Name:     s.options.Name,
		Value:    id,
		Path:     s.options.Path,
		Secure:   s.options.Secure,
		HttpOnly: s.options.HTTPOnly,
		SameSite: s.options.SameSite,
	}
	if s.options.TTL > 0 {
		cookie.MaxAge = int(s.options.TTL.Seconds())
		cookie.Expires = time.Now().Add(s.options.TTL)
	}

	http.SetCookie(w, cookie)
}

func (s *RedisStore) clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     s.options.Name,
		Value:    "",
		Path:     s.options.Path,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   s.options.Secure,
		HttpOnly: s.options.HTTPOnly,
		SameSite: s.options.SameSite,
	}
	if s.options.TTL > 0 {
		cookie.MaxAge = -1
	}
	http.SetCookie(w, cookie)
}

func (s *RedisStore) key(id string) string {
	return s.options.Prefix + id
}

func (s *RedisStore) get(id string) ([]byte, error) {
	resp, err := s.client.Do("GET", s.key(id))
	if err != nil {
		return nil, err
	}
	value, ok := resp.([]byte)
	if !ok {
		return nil, errors.New("redis: invalid response")
	}
	return value, nil
}

func (s *RedisStore) set(id string, payload []byte) error {
	args := []string{"SET", s.key(id), string(payload)}
	if s.options.TTL > 0 {
		ms := s.options.TTL.Milliseconds()
		if ms <= 0 {
			ms = 1
		}
		args = append(args, "PX", strconv.FormatInt(ms, 10))
	}
	_, err := s.client.Do(args...)
	return err
}

func (s *RedisStore) del(id string) error {
	_, err := s.client.Do("DEL", s.key(id))
	return err
}
