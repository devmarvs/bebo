package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/router"
)

// Do executes a request against a handler.
func Do(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// MustStatus asserts the response status code.
func MustStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected status %d, got %d", status, rec.Code)
	}
}

// MustHeader asserts a response header value.
func MustHeader(t *testing.T, rec *httptest.ResponseRecorder, key, value string) {
	t.Helper()
	if got := rec.Header().Get(key); got != value {
		t.Fatalf("expected header %s=%q, got %q", key, value, got)
	}
}

// DecodeJSON decodes a JSON response into dst.
func DecodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

// MiddlewareCase describes a middleware test case.
type MiddlewareCase struct {
	Name       string
	Middleware []bebo.Middleware
	Handler    bebo.Handler
	Request    *http.Request
	Assert     func(t *testing.T, rec *httptest.ResponseRecorder, err error)
}

// RunMiddleware executes middleware with a handler and request.
func RunMiddleware(t *testing.T, middleware []bebo.Middleware, handler bebo.Handler, req *http.Request) (*httptest.ResponseRecorder, error) {
	t.Helper()
	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "/", nil)
	}

	app := bebo.New()
	rec := httptest.NewRecorder()
	ctx := bebo.NewContext(rec, req, router.Params{}, app)

	h := handler
	if h == nil {
		h = func(*bebo.Context) error { return nil }
	}
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	err := h(ctx)
	return rec, err
}

// RunMiddlewareCases executes middleware test cases in a table-driven style.
func RunMiddlewareCases(t *testing.T, cases []MiddlewareCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			h := tc.Handler
			if h == nil {
				h = func(*bebo.Context) error { return nil }
			}
			rec, err := RunMiddleware(t, tc.Middleware, h, tc.Request)
			if tc.Assert != nil {
				tc.Assert(t, rec, err)
			}
		})
	}
}
