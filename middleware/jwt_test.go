package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/auth"
	"github.com/devmarvs/bebo/testutil"
)

func TestJWTMiddlewareAuthenticates(t *testing.T) {
	key := auth.JWTKey{Secret: []byte("secret")}
	token, err := auth.SignHS256(key, map[string]any{"sub": "user-1"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	_, err = testutil.RunMiddleware(t, []bebo.Middleware{
		JWT(JWTOptions{Authenticator: auth.JWTAuthenticator{Key: key.Secret}}),
	}, func(ctx *bebo.Context) error {
		principal, ok := bebo.PrincipalFromContext(ctx)
		if !ok || principal.ID != "user-1" {
			t.Fatalf("expected principal to be set")
		}
		return nil
	}, req)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
}
