package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/testutil"
)

func TestRateLimitPoliciesMatch(t *testing.T) {
	policies, err := NewRateLimitPolicies([]RateLimitPolicy{
		{
			Method: http.MethodGet,
			Path:   "/reports/:id",
			Allow: func(_ *bebo.Context, _ string) (bool, error) {
				return false, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("policies: %v", err)
	}

	mw := policies.Middleware()
	req := httptest.NewRequest(http.MethodGet, "/reports/123", nil)
	_, err = testutil.RunMiddleware(t, []bebo.Middleware{mw}, nil, req)
	appErr := apperr.As(err)
	if appErr == nil || appErr.Code != apperr.CodeRateLimited {
		t.Fatalf("expected rate limited error, got %v", err)
	}
}

func TestRateLimitPoliciesSkip(t *testing.T) {
	policies, err := NewRateLimitPolicies([]RateLimitPolicy{
		{
			Method: http.MethodGet,
			Path:   "/reports/:id",
			Allow: func(_ *bebo.Context, _ string) (bool, error) {
				return false, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("policies: %v", err)
	}

	mw := policies.Middleware()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	_, err = testutil.RunMiddleware(t, []bebo.Middleware{mw}, nil, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
