package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
)

// ErrInvalidKey indicates a missing signing key.
var ErrInvalidKey = errors.New("invalid signing key")

// JWTKey represents a signing key with an optional key ID.
type JWTKey struct {
	ID     string
	Secret []byte
}

// JWTKeySet supports signing and verifying with rotating keys.
type JWTKeySet struct {
	Primary  JWTKey
	Fallback []JWTKey
}

// Keys returns the full ordered key set.
func (s JWTKeySet) Keys() []JWTKey {
	keys := make([]JWTKey, 0, 1+len(s.Fallback))
	if len(s.Primary.Secret) > 0 {
		keys = append(keys, s.Primary)
	}
	for _, key := range s.Fallback {
		if len(key.Secret) == 0 {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

// Lookup finds a key by ID.
func (s JWTKeySet) Lookup(id string) (JWTKey, bool) {
	if id == "" {
		return JWTKey{}, false
	}
	if s.Primary.ID == id {
		return s.Primary, len(s.Primary.Secret) > 0
	}
	for _, key := range s.Fallback {
		if key.ID == id && len(key.Secret) > 0 {
			return key, true
		}
	}
	return JWTKey{}, false
}

// Sign creates an HS256-signed token using the primary key.
func (s JWTKeySet) Sign(claims map[string]any) (string, error) {
	return SignHS256(s.Primary, claims)
}

// SignHS256 creates an HS256-signed JWT for the provided claims.
func SignHS256(key JWTKey, claims map[string]any) (string, error) {
	if len(key.Secret) == 0 {
		return "", ErrInvalidKey
	}
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	if key.ID != "" {
		header["kid"] = key.ID
	}

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

	sig := hmac.New(sha256.New, key.Secret)
	_, _ = sig.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(sig.Sum(nil))

	return message + "." + signature, nil
}
