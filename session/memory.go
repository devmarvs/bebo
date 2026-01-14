package session

import (
	"container/heap"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"
)

type memoryEntry struct {
	values    map[string]string
	expiresAt time.Time
}

type expirationEntry struct {
	id        string
	expiresAt time.Time
}

type expirationHeap []expirationEntry

func (h expirationHeap) Len() int {
	return len(h)
}

func (h expirationHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h expirationHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *expirationHeap) Push(x any) {
	*h = append(*h, x.(expirationEntry))
}

func (h *expirationHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// MemoryStore stores sessions in memory and uses a session ID cookie.
type MemoryStore struct {
	Name     string
	TTL      time.Duration
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite

	mu          sync.RWMutex
	sessions    map[string]memoryEntry
	expirations expirationHeap
}

// MemoryOption configures a MemoryStore.
type MemoryOption func(*MemoryStore)

// WithSessionPath sets the session cookie path.
func WithSessionPath(path string) MemoryOption {
	return func(store *MemoryStore) {
		store.Path = path
	}
}

// WithSessionSecure sets the session cookie Secure flag.
func WithSessionSecure(enabled bool) MemoryOption {
	return func(store *MemoryStore) {
		store.Secure = enabled
	}
}

// WithSessionHTTPOnly sets the session cookie HttpOnly flag.
func WithSessionHTTPOnly(enabled bool) MemoryOption {
	return func(store *MemoryStore) {
		store.HTTPOnly = enabled
	}
}

// WithSessionSameSite sets the session cookie SameSite flag.
func WithSessionSameSite(mode http.SameSite) MemoryOption {
	return func(store *MemoryStore) {
		store.SameSite = mode
	}
}

// NewMemoryStore creates an in-memory store.
func NewMemoryStore(name string, ttl time.Duration, options ...MemoryOption) *MemoryStore {
	store := &MemoryStore{
		Name:     name,
		TTL:      ttl,
		Path:     "/",
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
		sessions: map[string]memoryEntry{},
	}
	for _, opt := range options {
		opt(store)
	}
	return store
}

// Get loads a session from the request.
func (s *MemoryStore) Get(r *http.Request) (*Session, error) {
	values := map[string]string{}
	now := time.Now()
	s.maybeCleanup(now)

	cookie, err := r.Cookie(s.Name)
	if err != nil || cookie.Value == "" {
		id, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: id, Values: values, store: s, isNew: true}, nil
	}

	id := cookie.Value
	s.mu.RLock()
	entry, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok || s.isExpired(entry, now) {
		if ok {
			s.mu.Lock()
			entry, ok = s.sessions[id]
			if ok && s.isExpired(entry, now) {
				delete(s.sessions, id)
			}
			s.mu.Unlock()
		}
		newID, err := newSessionID()
		if err != nil {
			return nil, err
		}
		return &Session{ID: newID, Values: values, store: s, isNew: true}, nil
	}

	return &Session{ID: id, Values: copyValues(entry.values), store: s}, nil
}

// Save persists a session.
func (s *MemoryStore) Save(w http.ResponseWriter, session *Session) error {
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

	now := time.Now()
	entry := memoryEntry{values: copyValues(session.Values)}
	s.mu.Lock()
	s.cleanupExpiredLocked(now)
	if s.TTL > 0 {
		entry.expiresAt = now.Add(s.TTL)
		heap.Push(&s.expirations, expirationEntry{id: id, expiresAt: entry.expiresAt})
	}
	s.sessions[id] = entry
	s.mu.Unlock()

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
		cookie.Expires = time.Now().Add(s.TTL)
	}

	http.SetCookie(w, cookie)
	session.isNew = false
	return nil
}

// Clear removes a session.
func (s *MemoryStore) Clear(w http.ResponseWriter, session *Session) {
	if session != nil && session.ID != "" {
		s.mu.Lock()
		delete(s.sessions, session.ID)
		s.mu.Unlock()
	}

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
		session.ID = ""
	}
}

func (s *MemoryStore) maybeCleanup(now time.Time) {
	if s.TTL <= 0 {
		return
	}
	s.mu.RLock()
	needsCleanup := len(s.expirations) > 0 && !s.expirations[0].expiresAt.After(now)
	s.mu.RUnlock()
	if !needsCleanup {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(now)
}

func (s *MemoryStore) cleanupExpiredLocked(now time.Time) {
	if s.TTL <= 0 {
		return
	}
	for len(s.expirations) > 0 {
		entry := s.expirations[0]
		if entry.expiresAt.After(now) {
			break
		}
		heap.Pop(&s.expirations)
		stored, ok := s.sessions[entry.id]
		if !ok {
			continue
		}
		if !stored.expiresAt.Equal(entry.expiresAt) {
			continue
		}
		if s.isExpired(stored, now) {
			delete(s.sessions, entry.id)
		}
	}
}

func (s *MemoryStore) isExpired(entry memoryEntry, now time.Time) bool {
	return !entry.expiresAt.IsZero() && !now.Before(entry.expiresAt)
}

func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func copyValues(values map[string]string) map[string]string {
	copy := make(map[string]string, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return copy
}
