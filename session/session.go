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

// Session represents cookie-backed session data.
type Session struct {
	Values map[string]string
	store  *CookieStore
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
		return &Session{Values: values, store: s}, nil
	}

	decoded, err := decode(cookie.Value, s.Keys)
	if err != nil {
		return &Session{Values: values, store: s}, ErrInvalidCookie
	}

	return &Session{Values: decoded, store: s}, nil
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

// Save writes the session cookie.
func (s *Session) Save(w http.ResponseWriter) error {
	if s.store == nil {
		return errors.New("session store missing")
	}

	value, err := encode(s.Values, s.store.Keys)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     s.store.Name,
		Value:    value,
		Path:     s.store.Path,
		Secure:   s.store.Secure,
		HttpOnly: s.store.HTTPOnly,
		SameSite: s.store.SameSite,
	}

	if s.store.MaxAge > 0 {
		cookie.MaxAge = int(s.store.MaxAge.Seconds())
		cookie.Expires = time.Now().Add(s.store.MaxAge)
	}

	http.SetCookie(w, cookie)
	return nil
}

// Clear expires the session cookie.
func (s *Session) Clear(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     s.store.Name,
		Value:    "",
		Path:     s.store.Path,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   s.store.Secure,
		HttpOnly: s.store.HTTPOnly,
		SameSite: s.store.SameSite,
	}
	http.SetCookie(w, cookie)
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
