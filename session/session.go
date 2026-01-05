package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	// ErrInvalidCookie indicates an invalid or tampered session cookie.
	ErrInvalidCookie = errors.New("invalid session cookie")
)

// Store defines a session persistence backend.
type Store interface {
	Get(*http.Request) (*Session, error)
	Save(http.ResponseWriter, *Session) error
	Clear(http.ResponseWriter, *Session)
}

// CookieStore stores sessions in a signed cookie.
type CookieStore struct {
	Name     string
	Keys     [][]byte
	Path     string
	MaxAge   time.Duration
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

// Session represents session data.
type Session struct {
	ID     string
	Values map[string]string
	store  Store
	isNew  bool
}

// IsNew reports whether the session was newly created for this request.
func (s *Session) IsNew() bool {
	return s.isNew
}

// NewCookieStore creates a cookie store with key rotation support.
func NewCookieStore(name string, key []byte, oldKeys ...[]byte) *CookieStore {
	keys := make([][]byte, 0, 1+len(oldKeys))
	keys = append(keys, key)
	keys = append(keys, oldKeys...)
	return &CookieStore{
		Name:     name,
		Keys:     keys,
		Path:     "/",
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// Get loads a session from the request.
func (s *CookieStore) Get(r *http.Request) (*Session, error) {
	values := map[string]string{}
	cookie, err := r.Cookie(s.Name)
	if err != nil {
		return &Session{Values: values, store: s, isNew: true}, nil
	}

	decoded, err := decode(cookie.Value, s.Keys)
	if err != nil {
		return &Session{Values: values, store: s, isNew: true}, ErrInvalidCookie
	}

	return &Session{Values: decoded, store: s}, nil
}

// Save writes the session cookie.
func (s *CookieStore) Save(w http.ResponseWriter, session *Session) error {
	if session == nil {
		return errors.New("session missing")
	}
	value, err := encode(session.Values, s.Keys)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     s.Name,
		Value:    value,
		Path:     s.Path,
		Secure:   s.Secure,
		HttpOnly: s.HTTPOnly,
		SameSite: s.SameSite,
	}

	if s.MaxAge > 0 {
		cookie.MaxAge = int(s.MaxAge.Seconds())
		cookie.Expires = time.Now().Add(s.MaxAge)
	}

	http.SetCookie(w, cookie)
	session.isNew = false
	return nil
}

// Clear expires the session cookie.
func (s *CookieStore) Clear(w http.ResponseWriter, session *Session) {
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
	http.SetCookie(w, cookie)
	if session != nil {
		session.Values = map[string]string{}
		session.isNew = true
	}
}

// Set sets a key value.
func (s *Session) Set(key, value string) {
	s.Values[key] = value
}

// Get returns a value.
func (s *Session) Get(key string) string {
	return s.Values[key]
}

// Delete removes a key.
func (s *Session) Delete(key string) {
	delete(s.Values, key)
}

// Save writes the session to the configured store.
func (s *Session) Save(w http.ResponseWriter) error {
	if s.store == nil {
		return errors.New("session store missing")
	}
	return s.store.Save(w, s)
}

// Clear clears the session from the configured store.
func (s *Session) Clear(w http.ResponseWriter) {
	if s.store == nil {
		return
	}
	s.store.Clear(w, s)
}

func encode(values map[string]string, keys [][]byte) (string, error) {
	if len(keys) == 0 || len(keys[0]) == 0 {
		return "", errors.New("session key required")
	}

	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	sig := sign(payload, keys[0])
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	encodedSig := base64.RawURLEncoding.EncodeToString(sig)
	return encodedPayload + "." + encodedSig, nil
}

func decode(value string, keys [][]byte) (map[string]string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidCookie
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidCookie
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidCookie
	}

	valid := false
	for _, key := range keys {
		if len(key) == 0 {
			continue
		}
		if hmac.Equal(signature, sign(payload, key)) {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidCookie
	}

	values := map[string]string{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func sign(payload []byte, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(payload)
	return h.Sum(nil)
}
