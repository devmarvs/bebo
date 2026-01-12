package httpclient

import (
	"errors"
	"net/http"

	"github.com/devmarvs/bebo"
)

// MetadataRoundTripper injects request metadata headers into outgoing requests.
type MetadataRoundTripper struct {
	Base http.RoundTripper
}

// RoundTrip executes the request with metadata propagation.
func (m *MetadataRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := m.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if req == nil {
		return nil, errors.New("request is nil")
	}

	metadata := bebo.RequestMetadataFromRequest(req)
	bebo.InjectRequestMetadataIfMissing(req, metadata)
	return base.RoundTrip(req)
}
