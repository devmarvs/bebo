package flash

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo/session"
)

func TestFlashStorePop(t *testing.T) {
	cookieStore := session.NewCookieStore("flash", []byte("secret"))
	store := New(cookieStore)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	if err := store.Add(rec, req, Message{Type: "success", Text: "saved"}); err != nil {
		t.Fatalf("add flash: %v", err)
	}

	resp := rec.Result()
	defer resp.Body.Close()

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	rec2 := httptest.NewRecorder()

	messages, err := store.Pop(rec2, req2)
	if err != nil {
		t.Fatalf("pop flash: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Text != "saved" {
		t.Fatalf("expected message text %q, got %q", "saved", messages[0].Text)
	}

	resp2 := rec2.Result()
	defer resp2.Body.Close()

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp2.Cookies() {
		req3.AddCookie(cookie)
	}

	messages, err = store.Peek(req3)
	if err != nil {
		t.Fatalf("peek flash: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}
}
