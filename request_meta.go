package bebo

import (
	"context"
	"net/http"
	"strings"
)

const (
	TraceparentHeader = "traceparent"
	TracestateHeader  = "tracestate"
)

// RequestMetadata holds request-scoped identifiers.
type RequestMetadata struct {
	RequestID   string
	Traceparent string
	Tracestate  string
}

type requestMetadataKey struct{}

// WithRequestMetadata stores metadata in a context.
func WithRequestMetadata(ctx context.Context, metadata RequestMetadata) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestMetadataKey{}, metadata)
}

// RequestMetadataFromContext returns metadata stored in a context.
func RequestMetadataFromContext(ctx context.Context) RequestMetadata {
	if ctx == nil {
		return RequestMetadata{}
	}
	value := ctx.Value(requestMetadataKey{})
	metadata, ok := value.(RequestMetadata)
	if !ok {
		return RequestMetadata{}
	}
	return metadata
}

// RequestMetadataFromRequest returns metadata from the request and context.
func RequestMetadataFromRequest(r *http.Request) RequestMetadata {
	if r == nil {
		return RequestMetadata{}
	}
	metadata := RequestMetadataFromContext(r.Context())
	if metadata.RequestID == "" {
		metadata.RequestID = RequestIDFromHeader(r)
	}
	if metadata.Traceparent == "" {
		metadata.Traceparent = r.Header.Get(TraceparentHeader)
	}
	if metadata.Tracestate == "" {
		metadata.Tracestate = r.Header.Get(TracestateHeader)
	}
	return metadata
}

// InjectRequestMetadata sets request metadata headers.
func InjectRequestMetadata(r *http.Request, metadata RequestMetadata) {
	if r == nil {
		return
	}
	if metadata.RequestID != "" {
		r.Header.Set(RequestIDHeader, metadata.RequestID)
	}
	if metadata.Traceparent != "" {
		r.Header.Set(TraceparentHeader, metadata.Traceparent)
	}
	if metadata.Tracestate != "" {
		r.Header.Set(TracestateHeader, metadata.Tracestate)
	}
}

// TraceIDs parses trace and span ids from traceparent.
func TraceIDs(traceparent string) (string, string, bool) {
	if traceparent == "" {
		return "", "", false
	}
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return "", "", false
	}
	traceID := strings.ToLower(parts[1])
	spanID := strings.ToLower(parts[2])
	if len(traceID) != 32 || len(spanID) != 16 {
		return "", "", false
	}
	if !isHex(traceID) || !isHex(spanID) {
		return "", "", false
	}
	if isAllZero(traceID) || isAllZero(spanID) {
		return "", "", false
	}
	return traceID, spanID, true
}

func isHex(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			continue
		}
		if r >= 'a' && r <= 'f' {
			continue
		}
		return false
	}
	return true
}

func isAllZero(value string) bool {
	for _, r := range value {
		if r != '0' {
			return false
		}
	}
	return true
}
