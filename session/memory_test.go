package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMemoryStoreLifecycle(t *testing.T) {
	store := NewMemoryStore("bebo_session", time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, err := store.Get(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !sess.IsNew() {
		t.Fatalf("expected new session")
	}

	sess.Set("user", "123")
	rec := httptest.NewRecorder()
	if err := sess.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(cookies[0])
	sess2, err := store.Get(req2)
	if err != nil {
		t.Fatalf("get stored: %v", err)
	}
	if sess2.Get("user") != "123" {
		t.Fatalf("expected stored value")
	}

	rec2 := httptest.NewRecorder()
	sess2.Clear(rec2)

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.AddCookie(cookies[0])
	sess3, err := store.Get(req3)
	if err != nil {
		t.Fatalf("get after clear: %v", err)
	}
	if !sess3.IsNew() {
		t.Fatalf("expected new session after clear")
	}
}

func TestMemoryStoreCleanupRemovesExpired(t *testing.T) {
	store := NewMemoryStore("bebo_session", 20*time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, err := store.Get(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	sess.Set("user", "123")
	rec := httptest.NewRecorder()
	if err := sess.Save(rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := len(store.sessions); got != 1 {
		t.Fatalf("expected 1 session, got %d", got)
	}

	time.Sleep(30 * time.Millisecond)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := store.Get(req2); err != nil {
		t.Fatalf("get after ttl: %v", err)
	}
	if got := len(store.sessions); got != 0 {
		t.Fatalf("expected expired session cleanup, got %d", got)
	}
}
