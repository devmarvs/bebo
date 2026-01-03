package bebo

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const RequestIDHeader = "X-Request-ID"

// NewRequestID generates a new request id.
func NewRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

// RequestIDFromHeader returns the request id from headers.
func RequestIDFromHeader(r *http.Request) string {
	return r.Header.Get(RequestIDHeader)
}
