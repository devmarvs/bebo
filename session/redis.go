package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var errRedisNil = errors.New("redis: nil")

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
	return &RedisStore{options: cfg}
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
	if errors.Is(err, errRedisNil) {
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
	var payload []byte
	return payload, s.withConn(func(conn *redisConn) error {
		resp, err := conn.do("GET", s.key(id))
		if err != nil {
			return err
		}
		if resp == nil {
			return errRedisNil
		}
		value, ok := resp.([]byte)
		if !ok {
			return errors.New("redis: invalid response")
		}
		payload = value
		return nil
	})
}

func (s *RedisStore) set(id string, payload []byte) error {
	return s.withConn(func(conn *redisConn) error {
		args := []string{"SET", s.key(id), string(payload)}
		if s.options.TTL > 0 {
			ms := s.options.TTL.Milliseconds()
			if ms <= 0 {
				ms = 1
			}
			args = append(args, "PX", strconv.FormatInt(ms, 10))
		}
		_, err := conn.do(args...)
		return err
	})
}

func (s *RedisStore) del(id string) error {
	return s.withConn(func(conn *redisConn) error {
		_, err := conn.do("DEL", s.key(id))
		return err
	})
}

func (s *RedisStore) withConn(fn func(*redisConn) error) error {
	conn, err := s.dial()
	if err != nil {
		return err
	}
	defer conn.close()
	return fn(conn)
}

func (s *RedisStore) dial() (*redisConn, error) {
	netConn, err := net.DialTimeout(s.options.Network, s.options.Address, s.options.DialTimeout)
	if err != nil {
		return nil, err
	}
	conn := &redisConn{
		conn:         netConn,
		rw:           bufio.NewReadWriter(bufio.NewReader(netConn), bufio.NewWriter(netConn)),
		readTimeout:  s.options.ReadTimeout,
		writeTimeout: s.options.WriteTimeout,
	}

	if s.options.Password != "" {
		args := []string{"AUTH"}
		if s.options.Username != "" {
			args = append(args, s.options.Username, s.options.Password)
		} else {
			args = append(args, s.options.Password)
		}
		if _, err := conn.do(args...); err != nil {
			conn.close()
			return nil, err
		}
	}
	if s.options.DB > 0 {
		if _, err := conn.do("SELECT", strconv.Itoa(s.options.DB)); err != nil {
			conn.close()
			return nil, err
		}
	}

	return conn, nil
}

type redisConn struct {
	conn         net.Conn
	rw           *bufio.ReadWriter
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (c *redisConn) do(args ...string) (any, error) {
	if err := c.writeCommand(args); err != nil {
		return nil, err
	}
	return c.readResponse()
}

func (c *redisConn) close() {
	_ = c.conn.Close()
}

func (c *redisConn) writeCommand(args []string) error {
	if c.writeTimeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	if _, err := fmt.Fprintf(c.rw, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(c.rw, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return c.rw.Flush()
}

func (c *redisConn) readResponse() (any, error) {
	if c.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
	line, err := c.rw.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if line == "" {
		return nil, errors.New("redis: empty response")
	}

	switch line[0] {
	case '+':
		return line[1:], nil
	case '-':
		return nil, errors.New(line[1:])
	case ':':
		value, err := strconv.ParseInt(line[1:], 10, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case '$':
		length, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(c.rw, buf); err != nil {
			return nil, err
		}
		return buf[:length], nil
	case '*':
		count, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		items := make([]any, 0, count)
		for i := 0; i < count; i++ {
			item, err := c.readResponse()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, errors.New("redis: unknown response")
	}
}
