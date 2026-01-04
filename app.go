package bebo

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/config"
	"github.com/devmarvs/bebo/logging"
	"github.com/devmarvs/bebo/render"
	"github.com/devmarvs/bebo/router"
)

// Handler handles a request and returns an error for centralized handling.
type Handler func(*Context) error

// Middleware wraps a handler with additional behavior.
type Middleware func(Handler) Handler

// ErrorHandler processes errors returned by handlers.
type ErrorHandler func(*Context, error)

type routeEntry struct {
	method     string
	pattern    string
	handler    Handler
	middleware []Middleware
}

// App is the main framework entrypoint.
type App struct {
	router       *router.Router
	routes       map[router.RouteID]*routeEntry
	middleware   []Middleware
	renderer     *render.Engine
	logger       *slog.Logger
	config       config.Config
	templateOpts render.Options
	errorHandler ErrorHandler
}

// Option customizes the app instance.
type Option func(*App)

// New creates a new App with defaults.
func New(options ...Option) *App {
	cfg := config.Default()

	app := &App{
		router:       router.New(),
		routes:       make(map[router.RouteID]*routeEntry),
		renderer:     nil,
		logger:       nil,
		config:       cfg,
		templateOpts: render.Options{Layout: cfg.LayoutTemplate, Reload: cfg.TemplateReload},
		errorHandler: defaultErrorHandler,
	}

	for _, opt := range options {
		opt(app)
	}

	if app.logger == nil {
		app.logger = logging.NewLogger(logging.Options{Level: app.config.LogLevel, Format: app.config.LogFormat})
	}

	if app.renderer == nil && app.config.TemplatesDir != "" {
		engine, err := render.NewEngineWithOptions(app.config.TemplatesDir, app.templateOpts)
		if err != nil {
			app.logger.Error("template load failed", slog.String("error", err.Error()))
		} else {
			app.renderer = engine
		}
	}

	return app
}

// WithConfig overrides the default config.
func WithConfig(cfg config.Config) Option {
	return func(app *App) {
		app.config = cfg
		app.templateOpts.Layout = cfg.LayoutTemplate
		app.templateOpts.Reload = cfg.TemplateReload
	}
}

// WithLogger uses a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(app *App) {
		app.logger = logger
	}
}

// WithRenderer sets a custom template engine.
func WithRenderer(engine *render.Engine) Option {
	return func(app *App) {
		app.renderer = engine
	}
}

// WithTemplateFuncs registers template functions for the built-in renderer.
func WithTemplateFuncs(funcs render.FuncMap) Option {
	return func(app *App) {
		if app.renderer != nil {
			_ = app.renderer.AddFuncs(funcs)
			return
		}
		if app.templateOpts.Funcs == nil {
			app.templateOpts.Funcs = render.FuncMap{}
		}
		for key, fn := range funcs {
			app.templateOpts.Funcs[key] = fn
		}
	}
}

// WithTemplateReload enables template reloading for development.
func WithTemplateReload(enabled bool) Option {
	return func(app *App) {
		app.templateOpts.Reload = enabled
	}
}

// WithErrorHandler overrides the default error handler.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(app *App) {
		app.errorHandler = handler
	}
}

// Use registers global middleware.
func (a *App) Use(middleware ...Middleware) {
	a.middleware = append(a.middleware, middleware...)
}

// GET registers a GET route.
func (a *App) GET(path string, handler Handler, middleware ...Middleware) {
	a.handle(http.MethodGet, path, handler, middleware...)
}

// POST registers a POST route.
func (a *App) POST(path string, handler Handler, middleware ...Middleware) {
	a.handle(http.MethodPost, path, handler, middleware...)
}

// PUT registers a PUT route.
func (a *App) PUT(path string, handler Handler, middleware ...Middleware) {
	a.handle(http.MethodPut, path, handler, middleware...)
}

// PATCH registers a PATCH route.
func (a *App) PATCH(path string, handler Handler, middleware ...Middleware) {
	a.handle(http.MethodPatch, path, handler, middleware...)
}

// DELETE registers a DELETE route.
func (a *App) DELETE(path string, handler Handler, middleware ...Middleware) {
	a.handle(http.MethodDelete, path, handler, middleware...)
}

// Handle registers a route for an arbitrary method.
func (a *App) Handle(method, path string, handler Handler, middleware ...Middleware) {
	a.handle(method, path, handler, middleware...)
}

func (a *App) handle(method, path string, handler Handler, middleware ...Middleware) {
	id, err := a.router.Add(method, path)
	if err != nil {
		a.logger.Error("route registration failed", slog.String("method", method), slog.String("path", path), slog.String("error", err.Error()))
		return
	}

	a.routes[id] = &routeEntry{
		method:     method,
		pattern:    path,
		handler:    handler,
		middleware: middleware,
	}
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, params, ok := a.router.Match(r.Method, r.URL.Path)
	if !ok {
		ctx := NewContext(w, r, router.Params{}, a)
		handler := func(ctx *Context) error {
			return apperr.New(apperr.CodeNotFound, http.StatusNotFound, "not found", nil)
		}
		for i := len(a.middleware) - 1; i >= 0; i-- {
			handler = a.middleware[i](handler)
		}
		if err := handler(ctx); err != nil {
			a.errorHandler(ctx, err)
		}
		return
	}

	entry := a.routes[id]
	ctx := NewContext(w, r, params, a)

	h := entry.handler
	for i := len(entry.middleware) - 1; i >= 0; i-- {
		h = entry.middleware[i](h)
	}
	for i := len(a.middleware) - 1; i >= 0; i-- {
		h = a.middleware[i](h)
	}

	if err := h(ctx); err != nil {
		a.errorHandler(ctx, err)
	}
}

// ListenAndServe starts the HTTP server using config values.
func (a *App) ListenAndServe() error {
	server := a.newServer()
	a.logger.Info("server starting", slog.String("address", a.config.Address))

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// ListenAndServeTLS starts the HTTPS server using config values.
func (a *App) ListenAndServeTLS(certFile, keyFile string) error {
	server := a.newServer()
	a.logger.Info("server starting", slog.String("address", a.config.Address))

	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Run starts the server and shuts down when the context is canceled.
func (a *App) Run(ctx context.Context) error {
	server := a.newServer()
	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("server starting", slog.String("address", a.config.Address))
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// RunWithSignals starts the server and handles SIGINT/SIGTERM for shutdown.
func (a *App) RunWithSignals() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.Run(ctx)
}

func defaultErrorHandler(ctx *Context, err error) {
	appErr := apperr.As(err)
	status := http.StatusInternalServerError
	code := apperr.CodeInternal
	message := "internal server error"

	if appErr != nil {
		status = appErr.Status
		code = appErr.Code
		message = appErr.Message
	}

	ctx.Logger().Error("request failed",
		slog.String("code", code),
		slog.String("error", err.Error()),
	)

	if wantsJSON(ctx.Request) {
		_ = ctx.JSON(status, map[string]any{
			"error": map[string]any{
				"code":    code,
				"message": message,
			},
		})
		return
	}

	_ = ctx.Text(status, message)
}

func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept == "" || strings.Contains(strings.ToLower(accept), "application/json")
}

// ShutdownTimeout returns the configured graceful shutdown timeout.
func (a *App) ShutdownTimeout() time.Duration {
	return a.config.ShutdownTimeout
}

func (a *App) newServer() *http.Server {
	return &http.Server{
		Addr:              a.config.Address,
		Handler:           a,
		ReadTimeout:       a.config.ReadTimeout,
		WriteTimeout:      a.config.WriteTimeout,
		IdleTimeout:       a.config.IdleTimeout,
		ReadHeaderTimeout: a.config.ReadHeaderTimeout,
		MaxHeaderBytes:    a.config.MaxHeaderBytes,
	}
}
