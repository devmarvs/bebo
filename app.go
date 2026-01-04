package bebo

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/devmarvs/bebo/apperr"
	"github.com/devmarvs/bebo/config"
	"github.com/devmarvs/bebo/logging"
	"github.com/devmarvs/bebo/render"
	"github.com/devmarvs/bebo/router"
	"github.com/devmarvs/bebo/validate"
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
	name       string
	timeout    time.Duration
}

// RouteInfo describes a named route.
type RouteInfo struct {
	Name    string
	Method  string
	Pattern string
}

// App is the main framework entrypoint.
type App struct {
	router       *router.Router
	routes       map[router.RouteID]*routeEntry
	routesByName map[string]router.RouteID
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
		routesByName: make(map[string]router.RouteID),
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

// Route registers a route with options.
func (a *App) Route(method, path string, handler Handler, options ...RouteOption) {
	a.handleWithOptions(method, path, handler, nil, options...)
}

// GET registers a GET route.
func (a *App) GET(path string, handler Handler, middleware ...Middleware) {
	a.handleWithOptions(http.MethodGet, path, handler, middleware)
}

// POST registers a POST route.
func (a *App) POST(path string, handler Handler, middleware ...Middleware) {
	a.handleWithOptions(http.MethodPost, path, handler, middleware)
}

// PUT registers a PUT route.
func (a *App) PUT(path string, handler Handler, middleware ...Middleware) {
	a.handleWithOptions(http.MethodPut, path, handler, middleware)
}

// PATCH registers a PATCH route.
func (a *App) PATCH(path string, handler Handler, middleware ...Middleware) {
	a.handleWithOptions(http.MethodPatch, path, handler, middleware)
}

// DELETE registers a DELETE route.
func (a *App) DELETE(path string, handler Handler, middleware ...Middleware) {
	a.handleWithOptions(http.MethodDelete, path, handler, middleware)
}

// Handle registers a route for an arbitrary method.
func (a *App) Handle(method, path string, handler Handler, middleware ...Middleware) {
	a.handleWithOptions(method, path, handler, middleware)
}

// RouteInfo returns metadata for a named route.
func (a *App) RouteInfo(name string) (RouteInfo, bool) {
	id, ok := a.routesByName[name]
	if !ok {
		return RouteInfo{}, false
	}
	entry, ok := a.routes[id]
	if !ok {
		return RouteInfo{}, false
	}
	return RouteInfo{Name: entry.name, Method: entry.method, Pattern: entry.pattern}, true
}

// Routes returns all named routes.
func (a *App) Routes() []RouteInfo {
	items := make([]RouteInfo, 0, len(a.routesByName))
	for name := range a.routesByName {
		if info, ok := a.RouteInfo(name); ok {
			items = append(items, info)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Pattern < items[j].Pattern
		}
		return items[i].Name < items[j].Name
	})
	return items
}

// Path builds a URL path from a named route and params.
func (a *App) Path(name string, params map[string]string) (string, bool) {
	info, ok := a.RouteInfo(name)
	if !ok {
		return "", false
	}
	return buildPath(info.Pattern, params)
}

func (a *App) handleWithOptions(method, path string, handler Handler, middleware []Middleware, options ...RouteOption) {
	cfg := routeConfig{}
	for _, opt := range options {
		opt(&cfg)
	}

	if cfg.name != "" {
		if _, exists := a.routesByName[cfg.name]; exists {
			a.logger.Error("route name already registered", slog.String("name", cfg.name))
			return
		}
	}

	id, err := a.router.Add(method, path)
	if err != nil {
		a.logger.Error("route registration failed", slog.String("method", method), slog.String("path", path), slog.String("error", err.Error()))
		return
	}

	combined := append([]Middleware{}, middleware...)
	combined = append(combined, cfg.middleware...)

	a.routes[id] = &routeEntry{
		method:     method,
		pattern:    path,
		handler:    handler,
		middleware: combined,
		name:       cfg.name,
		timeout:    cfg.timeout,
	}

	if cfg.name != "" {
		a.routesByName[cfg.name] = id
	}
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, params, ok := a.router.Match(r.Method, r.URL.Path)
	if !ok {
		allowed := a.router.Allowed(r.URL.Path)
		ctx := NewContext(w, r, router.Params{}, a)
		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			a.runWithMiddleware(ctx, func(ctx *Context) error {
				return apperr.MethodNotAllowed("method not allowed", nil)
			})
			return
		}
		a.runWithMiddleware(ctx, func(ctx *Context) error {
			return apperr.NotFound("not found", nil)
		})
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
	if entry.timeout > 0 {
		h = TimeoutHandler(h, entry.timeout)
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

func (a *App) runWithMiddleware(ctx *Context, handler Handler) {
	for i := len(a.middleware) - 1; i >= 0; i-- {
		handler = a.middleware[i](handler)
	}
	if err := handler(ctx); err != nil {
		a.errorHandler(ctx, err)
	}
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

	var fields []validate.FieldError
	if validationErrors, ok := validate.As(err); ok {
		fields = validationErrors.Fields
	}

	if wantsJSON(ctx.Request) {
		payload := map[string]any{
			"error": map[string]any{
				"code":    code,
				"message": message,
			},
		}
		if len(fields) > 0 {
			payload["error"].(map[string]any)["fields"] = fields
		}
		_ = ctx.JSON(status, payload)
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
