package middleware

import (
	"net/http"
	"strings"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

// MethodOverrideOptions configures form method overrides.
type MethodOverrideOptions struct {
	DisableDefaults bool
	HeaderName      string
	FormField       string
	AllowedMethods  []string
}

type methodOverrideConfig struct {
	headerName string
	formField  string
	allowed    map[string]struct{}
}

// MethodOverride enables HTML form method overrides via header or _method field.
func MethodOverride(options MethodOverrideOptions) bebo.PreMiddleware {
	cfg := normalizeMethodOverride(options)
	return func(ctx *bebo.Context) error {
		r := ctx.Request
		if r.Method != http.MethodPost {
			return nil
		}

		override := strings.TrimSpace(r.Header.Get(cfg.headerName))
		if override == "" && isFormRequest(r) {
			_ = r.ParseForm()
			override = r.Form.Get(cfg.formField)
		}
		if override == "" {
			return nil
		}

		override = strings.ToUpper(strings.TrimSpace(override))
		if _, ok := cfg.allowed[override]; !ok {
			return apperr.BadRequest("method override not allowed", nil)
		}

		r.Method = override
		return nil
	}
}

func normalizeMethodOverride(options MethodOverrideOptions) methodOverrideConfig {
	cfg := methodOverrideConfig{}
	if options.HeaderName != "" {
		cfg.headerName = options.HeaderName
	} else {
		cfg.headerName = "X-HTTP-Method-Override"
	}
	if options.FormField != "" {
		cfg.formField = options.FormField
	} else {
		cfg.formField = "_method"
	}

	allowed := options.AllowedMethods
	if len(allowed) == 0 && !options.DisableDefaults {
		allowed = []string{http.MethodPut, http.MethodPatch, http.MethodDelete}
	}
	cfg.allowed = make(map[string]struct{}, len(allowed))
	for _, method := range allowed {
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" {
			continue
		}
		cfg.allowed[method] = struct{}{}
	}
	return cfg
}
