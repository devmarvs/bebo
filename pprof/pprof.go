package pprof

import (
	"errors"
	"net/http"
	netpprof "net/http/pprof"
	"strings"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/middleware"
)

// Option customizes pprof registration.
type Option func(*options)

type options struct {
	prefix              string
	authorizer          bebo.Authorizer
	unauthorizedMessage string
	forbiddenMessage    string
}

// WithPrefix overrides the default pprof prefix.
func WithPrefix(prefix string) Option {
	return func(o *options) {
		o.prefix = prefix
	}
}

// WithAuthorizer sets a custom authorizer for pprof routes.
func WithAuthorizer(authorizer bebo.Authorizer) Option {
	return func(o *options) {
		o.authorizer = authorizer
	}
}

// WithUnauthorizedMessage sets the unauthorized error message.
func WithUnauthorizedMessage(message string) Option {
	return func(o *options) {
		o.unauthorizedMessage = message
	}
}

// WithForbiddenMessage sets the forbidden error message.
func WithForbiddenMessage(message string) Option {
	return func(o *options) {
		o.forbiddenMessage = message
	}
}

// Register mounts authenticated pprof routes on the app.
func Register(app *bebo.App, auth bebo.Authenticator, opts ...Option) error {
	if app == nil {
		return errors.New("app is nil")
	}
	if auth == nil {
		return errors.New("authenticator is required")
	}

	cfg := options{
		prefix:              "/debug/pprof",
		unauthorizedMessage: "unauthorized",
		forbiddenMessage:    "forbidden",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.prefix = normalizePrefix(cfg.prefix)
	if cfg.prefix == "" {
		cfg.prefix = "/debug/pprof"
	}

	middlewareOpts := []middleware.AuthOption{
		middleware.AuthUnauthorizedMessage(cfg.unauthorizedMessage),
		middleware.AuthForbiddenMessage(cfg.forbiddenMessage),
	}

	group := app.Group(cfg.prefix, middleware.RequireAuthorization(auth, cfg.authorizer, middlewareOpts...))

	group.GET("/", handlerFromHTTP(netpprof.Index))
	group.GET("/cmdline", handlerFromHTTP(netpprof.Cmdline))
	group.GET("/profile", handlerFromHTTP(netpprof.Profile))
	group.GET("/symbol", handlerFromHTTP(netpprof.Symbol))
	group.POST("/symbol", handlerFromHTTP(netpprof.Symbol))
	group.GET("/trace", handlerFromHTTP(netpprof.Trace))
	group.GET("/:profile", func(ctx *bebo.Context) error {
		netpprof.Handler(ctx.Param("profile")).ServeHTTP(ctx.ResponseWriter, ctx.Request)
		return nil
	})

	return nil
}

func handlerFromHTTP(handler func(http.ResponseWriter, *http.Request)) bebo.Handler {
	return func(ctx *bebo.Context) error {
		handler(ctx.ResponseWriter, ctx.Request)
		return nil
	}
}

func normalizePrefix(prefix string) string {
	if prefix == "" {
		return prefix
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 {
		prefix = strings.TrimRight(prefix, "/")
	}
	return prefix
}
