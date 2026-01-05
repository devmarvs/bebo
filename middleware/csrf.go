package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

const csrfKey = "bebo.csrf"

// CSRFOptions configures CSRF behavior.
type CSRFOptions struct {
	DisableDefaults bool
	CookieName      string
	HeaderName      string
	FormField       string
	CookiePath      string
	CookieSecure    bool
	CookieHTTPOnly  bool
	CookieSameSite  http.SameSite
	TokenLength     int
	Rotate          bool
}

// CSRF protects against cross-site request forgery using a double-submit cookie.
func CSRF(options CSRFOptions) bebo.Middleware {
	cfg := normalizeCSRF(options)
	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			token, _ := csrfFromCookie(ctx.Request, cfg.CookieName)
			if token == "" {
				newToken, err := generateToken(cfg.TokenLength)
				if err != nil {
					return apperr.Internal("csrf token generation failed", err)
				}
				token = newToken
				setCSRFCookie(ctx.ResponseWriter, token, cfg)
			}

			ctx.Set(csrfKey, token)

			if isUnsafeMethod(ctx.Request.Method) {
				submitted := ctx.Request.Header.Get(cfg.HeaderName)
				if submitted == "" && isFormRequest(ctx.Request) {
					_ = ctx.Request.ParseForm()
					submitted = ctx.Request.Form.Get(cfg.FormField)
				}
				if submitted == "" || !secureCompare(submitted, token) {
					return apperr.Forbidden("csrf token invalid", nil)
				}
			}

			if cfg.Rotate {
				newToken, err := generateToken(cfg.TokenLength)
				if err != nil {
					return apperr.Internal("csrf token generation failed", err)
				}
				setCSRFCookie(ctx.ResponseWriter, newToken, cfg)
				ctx.Set(csrfKey, newToken)
			}

			return next(ctx)
		}
	}
}

// CSRFToken returns the request CSRF token.
func CSRFToken(ctx *bebo.Context) string {
	value, ok := ctx.Get(csrfKey)
	if !ok {
		return ""
	}
	if token, ok := value.(string); ok {
		return token
	}
	return ""
}

func normalizeCSRF(options CSRFOptions) CSRFOptions {
	if !options.DisableDefaults {
		if options.CookieName == "" {
			options.CookieName = "bebo_csrf"
		}
		if options.HeaderName == "" {
			options.HeaderName = "X-CSRF-Token"
		}
		if options.FormField == "" {
			options.FormField = "csrf_token"
		}
		if options.CookiePath == "" {
			options.CookiePath = "/"
		}
		if options.TokenLength <= 0 {
			options.TokenLength = 32
		}
		if options.CookieSameSite == 0 {
			options.CookieSameSite = http.SameSiteLaxMode
		}
		if !options.CookieHTTPOnly {
			options.CookieHTTPOnly = true
		}
	}
	return options
}

func csrfFromCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func setCSRFCookie(w http.ResponseWriter, token string, options CSRFOptions) {
	http.SetCookie(w, &http.Cookie{
		Name:     options.CookieName,
		Value:    token,
		Path:     options.CookiePath,
		Secure:   options.CookieSecure,
		HttpOnly: options.CookieHTTPOnly,
		SameSite: options.CookieSameSite,
	})
}

func generateToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func isFormRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/x-www-form-urlencoded") || strings.HasPrefix(contentType, "multipart/form-data")
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
