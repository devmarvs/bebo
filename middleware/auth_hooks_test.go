package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
)

type testAuth struct {
	principal *bebo.Principal
	err       error
}

func (t testAuth) Authenticate(ctx *bebo.Context) (*bebo.Principal, error) {
	return t.principal, t.err
}

func TestAuthHooks(t *testing.T) {
	calledBefore := false
	calledAfter := false
	var gotPrincipal *bebo.Principal

	app := bebo.New(bebo.WithAuthHooks(bebo.AuthHooks{
		BeforeAuthenticate: func(ctx *bebo.Context) { calledBefore = true },
		AfterAuthenticate: func(ctx *bebo.Context, principal *bebo.Principal, err error) {
			calledAfter = true
			gotPrincipal = principal
		},
	}))

	app.GET("/private", func(ctx *bebo.Context) error {
		return ctx.Text(http.StatusOK, "ok")
	}, RequireAuth(testAuth{principal: &bebo.Principal{ID: "user-1"}}))

	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Get(server.URL + "/private")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if !calledBefore {
		t.Fatalf("expected BeforeAuthenticate hook to be called")
	}
	if !calledAfter {
		t.Fatalf("expected AfterAuthenticate hook to be called")
	}
	if gotPrincipal == nil || gotPrincipal.ID != "user-1" {
		t.Fatalf("expected principal to be passed to hook")
	}
}
