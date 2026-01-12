package middleware

import (
	"net/http"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/session"
	"github.com/devmarvs/bebo/testutil"
)

type stubSessionStore struct {
	session *session.Session
	err     error
	cleared bool
}

func (s *stubSessionStore) Get(*http.Request) (*session.Session, error) {
	return s.session, s.err
}

func (s *stubSessionStore) Save(http.ResponseWriter, *session.Session) error {
	return nil
}

func (s *stubSessionStore) Clear(http.ResponseWriter, *session.Session) {
	s.cleared = true
}

func TestSessionMiddlewareStoresSession(t *testing.T) {
	store := &stubSessionStore{session: &session.Session{Values: map[string]string{}}}

	_, err := testutil.RunMiddleware(t, []bebo.Middleware{Session(store)}, func(ctx *bebo.Context) error {
		sess, ok := SessionFromContext(ctx)
		if !ok || sess == nil {
			t.Fatalf("expected session on context")
		}
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
}

func TestSessionMiddlewareClearsInvalidCookie(t *testing.T) {
	store := &stubSessionStore{
		session: &session.Session{Values: map[string]string{}},
		err:     session.ErrInvalidCookie,
	}

	_, err := testutil.RunMiddleware(t, []bebo.Middleware{Session(store)}, func(*bebo.Context) error {
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if !store.cleared {
		t.Fatalf("expected invalid cookie to be cleared")
	}
}
