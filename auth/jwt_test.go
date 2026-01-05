package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devmarvs/bebo"
)

func TestJWTAuthenticatorValid(t *testing.T) {
	key := []byte("secret")
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	token, err := buildToken(key, map[string]any{
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

	token, err := buildToken(key, map[string]any{
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

func buildToken(key []byte, claims map[string]any) (string, error) {
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	message := encodedHeader + "." + encodedPayload

	sig := hmac.New(sha256.New, key)
	_, _ = sig.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(sig.Sum(nil))

	return message + "." + signature, nil
}
