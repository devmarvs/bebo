package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/flash"
	"github.com/devmarvs/bebo/router"
	"github.com/devmarvs/bebo/session"
)

func TestTemplateDataFrom(t *testing.T) {
	cookieStore := session.NewCookieStore("flash", []byte("secret"))
	store := flash.New(cookieStore)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	if err := store.Add(rec, req, flash.Message{Type: "info", Text: "hello"}); err != nil {
		t.Fatalf("add flash: %v", err)
	}

	resp := rec.Result()
	defer resp.Body.Close()

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}
	rec2 := httptest.NewRecorder()

	app := bebo.New()
	ctx := bebo.NewContext(rec2, req2, router.Params{}, app)
	ctx.Set("bebo.csrf", "token")

	view, err := TemplateDataFrom(ctx, &store, map[string]string{"Title": "Hello"})
	if err != nil {
		t.Fatalf("template data: %v", err)
	}
	if view.CSRFToken != "token" {
		t.Fatalf("expected csrf token %q, got %q", "token", view.CSRFToken)
	}
	if len(view.Flash) != 1 {
		t.Fatalf("expected 1 flash message, got %d", len(view.Flash))
	}
}

func TestCSRFFieldEscapes(t *testing.T) {
	field := CSRFFieldNamed("csrf_token", "bad\"token")
	if !strings.Contains(string(field), "bad&#34;token") {
		t.Fatalf("expected csrf field to escape token")
	}
}
