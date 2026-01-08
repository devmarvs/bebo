package security

import (
	"net/http"
	"testing"
)

func TestCSPBuilder(t *testing.T) {
	policy := NewCSP().
		DefaultSrc("'self'").
		ScriptSrc("'self'", "cdn.example.com").
		UpgradeInsecureRequests()

	expected := "default-src 'self'; script-src 'self' cdn.example.com; upgrade-insecure-requests"
	if got := policy.String(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSecureCookieDefaults(t *testing.T) {
	cookie := NewSecureCookie("session", "value", CookieOptions{})
	if cookie.Path != "/" {
		t.Fatalf("expected path '/', got %q", cookie.Path)
	}
	if !cookie.Secure {
		t.Fatalf("expected secure cookie")
	}
	if !cookie.HttpOnly {
		t.Fatalf("expected HttpOnly cookie")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite Lax, got %v", cookie.SameSite)
	}
}

func TestSecureCookieDisableDefaults(t *testing.T) {
	cookie := NewSecureCookie("session", "value", CookieOptions{DisableDefaults: true})
	if cookie.Path != "" {
		t.Fatalf("expected empty path, got %q", cookie.Path)
	}
	if cookie.Secure {
		t.Fatalf("expected secure false")
	}
	if cookie.HttpOnly {
		t.Fatalf("expected HttpOnly false")
	}
	if cookie.SameSite != 0 {
		t.Fatalf("expected SameSite 0, got %v", cookie.SameSite)
	}
}
