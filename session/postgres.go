package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const DefaultPostgresTable = "bebo_sessions"

// PostgresOptions configures a Postgres store.
type PostgresOptions struct {
	DisableDefaults bool
	DB              *sql.DB
	Name            string
	Table           string
	TTL             time.Duration
	Timeout         time.Duration
	Path            string
	Secure          bool
	HTTPOnly        bool
	SameSite        http.SameSite
}

// PostgresStore stores sessions in PostgreSQL using a session ID cookie.
type PostgresStore struct {
	DB       *sql.DB
	Name     string
	Table    string
	TTL      time.Duration
	Timeout  time.Duration
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
	now      func() time.Time
}

// NewPostgresStore builds a Postgres-backed store.
func NewPostgresStore(options PostgresOptions) (*PostgresStore, error) {
	if options.DB == nil {
		return nil, errors.New("postgres db is required")
	}
	name := strings.TrimSpace(options.Name)
	if name == "" {
		name = "bebo_session"
	}
	table := strings.TrimSpace(options.Table)
	if table == "" {
		table = DefaultPostgresTable
	}
	if !validPostgresTable(table) {
		return nil, fmt.Errorf("invalid postgres table name: %s", table)
	}

	store := &PostgresStore{
		DB:       options.DB,
		Name:     name,
		Table:    table,
		TTL:      options.TTL,
		Timeout:  options.Timeout,
		Path:     options.Path,
		Secure:   options.Secure,
		HTTPOnly: options.HTTPOnly,
		SameSite: options.SameSite,
		now:      time.Now,
	}
	if !options.DisableDefaults {
		if store.Path == "" {
			store.Path = "/"
		}
		if !store.HTTPOnly {
			store.HTTPOnly = true
		}
		if store.SameSite == 0 {
			store.SameSite = http.SameSiteLaxMode
		}
	}
	return store, nil
}

// EnsureTable creates the session table if it does not exist.
func (s *PostgresStore) EnsureTable(ctx context.Context) error {
	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id TEXT PRIMARY KEY,
		data JSONB NOT NULL,
		expires_at TIMESTAMPTZ
	)`, s.Table)
	if _, err := s.DB.ExecContext(ctx, createTable); err != nil {
		return err
	}
	createIndex := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_expires_idx ON %s (expires_at)", s.Table, s.Table)
	_, err := s.DB.ExecContext(ctx, createIndex)
	return err
}

// Cleanup removes expired sessions.
func (s *PostgresStore) Cleanup(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE expires_at IS NOT NULL AND expires_at < NOW()", s.Table)
	_, err := s.DB.ExecContext(ctx, query)
	return err
}

// Get loads a session from the request.
func (s *PostgresStore) Get(r *http.Request) (*Session, error) {
	values := map[string]string{}
	cookie, err := r.Cookie(s.Name)
	if err != nil || cookie.Value == "" {
		id, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: id, Values: values, store: s, isNew: true}, nil
	}

	ctx, cancel := s.ctx()
	defer cancel()

	var payload []byte
	var expires sql.NullTime
	query := fmt.Sprintf("SELECT data, expires_at FROM %s WHERE id = $1", s.Table)
	err = s.DB.QueryRowContext(ctx, query, cookie.Value).Scan(&payload, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		id, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: id, Values: values, store: s, isNew: true}, nil
	}
	if err != nil {
		return nil, err
	}

	if expires.Valid && s.now().After(expires.Time) {
		_ = s.deleteByID(ctx, cookie.Value)
		id, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: id, Values: values, store: s, isNew: true}, nil
	}

	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}

	return &Session{ID: cookie.Value, Values: values, store: s}, nil
}

// Save persists a session.
func (s *PostgresStore) Save(w http.ResponseWriter, session *Session) error {
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

	var expires sql.NullTime
	if s.TTL > 0 {
		expires = sql.NullTime{Time: s.now().Add(s.TTL), Valid: true}
	}

	ctx, cancel := s.ctx()
	defer cancel()

	query := fmt.Sprintf(`INSERT INTO %s (id, data, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, expires_at = EXCLUDED.expires_at`, s.Table)
	if _, err := s.DB.ExecContext(ctx, query, id, payload, expires); err != nil {
		return err
	}

	s.setCookie(w, id)
	session.isNew = false
	return nil
}

// Clear removes a session.
func (s *PostgresStore) Clear(w http.ResponseWriter, session *Session) {
	if session != nil && session.ID != "" {
		ctx, cancel := s.ctx()
		defer cancel()
		_ = s.deleteByID(ctx, session.ID)
	}

	s.clearCookie(w)
	if session != nil {
		session.Values = map[string]string{}
		session.isNew = true
		session.ID = ""
	}
}

func (s *PostgresStore) deleteByID(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.Table)
	_, err := s.DB.ExecContext(ctx, query, id)
	return err
}

func (s *PostgresStore) setCookie(w http.ResponseWriter, id string) {
	cookie := &http.Cookie{
		Name:     s.Name,
		Value:    id,
		Path:     s.Path,
		Secure:   s.Secure,
		HttpOnly: s.HTTPOnly,
		SameSite: s.SameSite,
	}
	if s.TTL > 0 {
		cookie.MaxAge = int(s.TTL.Seconds())
		cookie.Expires = s.now().Add(s.TTL)
	}

	http.SetCookie(w, cookie)
}

func (s *PostgresStore) clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     s.Name,
		Value:    "",
		Path:     s.Path,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   s.Secure,
		HttpOnly: s.HTTPOnly,
		SameSite: s.SameSite,
	}
	if s.TTL > 0 {
		cookie.MaxAge = -1
	}
	http.SetCookie(w, cookie)
}

func (s *PostgresStore) ctx() (context.Context, context.CancelFunc) {
	if s.Timeout <= 0 {
		return context.Background(), func() {}
	}
	return context.WithTimeout(context.Background(), s.Timeout)
}

var tableNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+(\.[a-zA-Z0-9_]+)?$`)

func validPostgresTable(name string) bool {
	return tableNamePattern.MatchString(name)
}
