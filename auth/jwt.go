package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/devmarvs/bebo"
)

var (
	// ErrInvalidToken indicates a malformed or invalid token.
	ErrInvalidToken = errors.New("invalid token")
	// ErrUnsupportedAlg indicates an unsupported signing algorithm.
	ErrUnsupportedAlg = errors.New("unsupported jwt alg")
	// ErrExpiredToken indicates an expired token.
	ErrExpiredToken = errors.New("token expired")
	// ErrNotBefore indicates a token that is not valid yet.
	ErrNotBefore = errors.New("token not valid yet")
	// ErrInvalidIssuer indicates an issuer mismatch.
	ErrInvalidIssuer = errors.New("token issuer invalid")
	// ErrInvalidAudience indicates an audience mismatch.
	ErrInvalidAudience = errors.New("token audience invalid")
)

// JWTAuthenticator validates HS256 JWT tokens.
type JWTAuthenticator struct {
	Key       []byte
	Issuer    string
	Audience  string
	Header    string
	Scheme    string
	ClockSkew time.Duration
	Now       func() time.Time
}

// Authenticate validates a JWT bearer token from the request.
func (a JWTAuthenticator) Authenticate(ctx *bebo.Context) (*bebo.Principal, error) {
	if ctx == nil {
		return nil, ErrInvalidToken
	}

	token := extractToken(ctx.Request.Header.Get(headerName(a.Header)), a.Scheme)
	if token == "" {
		return nil, nil
	}

	if len(a.Key) == 0 {
		return nil, ErrInvalidToken
	}

	claims, err := a.verify(token)
	if err != nil {
		return nil, err
	}

	subject, _ := claims["sub"].(string)
	roles := parseRoles(claims["roles"])

	return &bebo.Principal{ID: subject, Roles: roles, Claims: claims}, nil
}

func (a JWTAuthenticator) verify(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrInvalidToken
	}
	if header.Alg != "HS256" {
		return nil, ErrUnsupportedAlg
	}

	payloadBytes, err := decodeSegment(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	sig, err := decodeSegment(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !verifySignature(a.Key, parts[0]+"."+parts[1], sig) {
		return nil, ErrInvalidToken
	}

	claims := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(payloadBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&claims); err != nil {
		return nil, ErrInvalidToken
	}

	now := time.Now()
	if a.Now != nil {
		now = a.Now()
	}

	if err := validateTiming(claims, now, a.ClockSkew); err != nil {
		return nil, err
	}
	if a.Issuer != "" {
		if issuer, _ := claims["iss"].(string); issuer != a.Issuer {
			return nil, ErrInvalidIssuer
		}
	}
	if a.Audience != "" {
		if !audienceMatches(claims["aud"], a.Audience) {
			return nil, ErrInvalidAudience
		}
	}

	return claims, nil
}

func headerName(value string) string {
	if value == "" {
		return "Authorization"
	}
	return value
}

func extractToken(value, scheme string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if scheme == "" {
		scheme = "Bearer"
	}
	prefix := strings.ToLower(scheme) + " "
	if strings.HasPrefix(strings.ToLower(value), prefix) {
		return strings.TrimSpace(value[len(prefix):])
	}
	return ""
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}

func verifySignature(key []byte, message string, sig []byte) bool {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(message))
	expected := mac.Sum(nil)
	return subtle.ConstantTimeCompare(expected, sig) == 1
}

func validateTiming(claims map[string]any, now time.Time, skew time.Duration) error {
	if exp := claimTime(claims["exp"]); !exp.IsZero() {
		if now.After(exp.Add(skew)) {
			return ErrExpiredToken
		}
	}
	if nbf := claimTime(claims["nbf"]); !nbf.IsZero() {
		if now.Add(skew).Before(nbf) {
			return ErrNotBefore
		}
	}
	return nil
}

func claimTime(value any) time.Time {
	switch typed := value.(type) {
	case json.Number:
		seconds, err := typed.Int64()
		if err != nil {
			return time.Time{}
		}
		return time.Unix(seconds, 0)
	case float64:
		return time.Unix(int64(typed), 0)
	case int64:
		return time.Unix(typed, 0)
	case int:
		return time.Unix(int64(typed), 0)
	default:
		return time.Time{}
	}
}

func parseRoles(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		roles := make([]string, 0, len(typed))
		for _, item := range typed {
			if role, ok := item.(string); ok {
				roles = append(roles, role)
			}
		}
		return roles
	case string:
		return []string{typed}
	default:
		return nil
	}
}

func audienceMatches(value any, expected string) bool {
	switch typed := value.(type) {
	case string:
		return typed == expected
	case []any:
		for _, item := range typed {
			if aud, ok := item.(string); ok && aud == expected {
				return true
			}
		}
	case []string:
		for _, aud := range typed {
			if aud == expected {
				return true
			}
		}
	}
	return false
}
