package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devmarvs/bebo"
)

func TestJWTAuthenticatorValid(t *testing.T) {
	key := []byte("secret")
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	token, err := buildToken(JWTKey{Secret: key}, map[string]any{
		"sub":   "user-1",
		"roles": []string{"admin"},
		"iss":   "bebo",
		"aud":   "api",
		"exp":   now.Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := bebo.NewContext(httptest.NewRecorder(), req, nil, bebo.New())

	auth := JWTAuthenticator{Key: key, Issuer: "bebo", Audience: "api", Now: func() time.Time { return now }}
	principal, err := auth.Authenticate(ctx)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal == nil {
		t.Fatalf("expected principal")
	}
	if principal.ID != "user-1" {
		t.Fatalf("expected subject user-1, got %s", principal.ID)
	}
	if len(principal.Roles) != 1 || principal.Roles[0] != "admin" {
		t.Fatalf("expected role admin")
	}
}

func TestJWTAuthenticatorExpired(t *testing.T) {
	key := []byte("secret")
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	token, err := buildToken(JWTKey{Secret: key}, map[string]any{
		"sub": "user-1",
		"exp": now.Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := bebo.NewContext(httptest.NewRecorder(), req, nil, bebo.New())

	auth := JWTAuthenticator{Key: key, Now: func() time.Time { return now }}
	principal, err := auth.Authenticate(ctx)
	if err != ErrExpiredToken {
		t.Fatalf("expected expired token error, got %v", err)
	}
	if principal != nil {
		t.Fatalf("expected no principal")
	}
}

func TestJWTAuthenticatorKeyRotation(t *testing.T) {
	newKey := []byte("new")
	oldKey := []byte("old")

	token, err := buildToken(JWTKey{Secret: oldKey}, map[string]any{"sub": "user-1"})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := bebo.NewContext(httptest.NewRecorder(), req, nil, bebo.New())

	auth := JWTAuthenticator{Key: newKey, Keys: [][]byte{oldKey}}
	principal, err := auth.Authenticate(ctx)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal == nil || principal.ID != "user-1" {
		t.Fatalf("expected principal from rotated key")
	}
}

func TestJWTAuthenticatorKeyID(t *testing.T) {
	keys := JWTKeySet{
		Primary: JWTKey{ID: "current", Secret: []byte("new")},
		Fallback: []JWTKey{
			{ID: "old", Secret: []byte("old")},
		},
	}

	token, err := buildToken(JWTKey{ID: "old", Secret: []byte("old")}, map[string]any{"sub": "user-1"})
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := bebo.NewContext(httptest.NewRecorder(), req, nil, bebo.New())

	auth := JWTAuthenticator{KeySet: &keys}
	principal, err := auth.Authenticate(ctx)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal == nil || principal.ID != "user-1" {
		t.Fatalf("expected principal for key id")
	}
}

func buildToken(key JWTKey, claims map[string]any) (string, error) {
	return SignHS256(key, claims)
}
