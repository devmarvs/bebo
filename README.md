# bebo

bebo is a batteries-included Go framework focused on building REST APIs and server-rendered web apps with a lightweight custom router. Desktop support is available via Fyne.

## Status
- v0.1 scaffold with custom router, middleware, JSON/HTML rendering, config defaults, static assets, and examples.
- Desktop helpers live in `desktop/` and depend on Fyne.

## Requirements
- Go 1.25 (as requested). If you are on a released Go toolchain, downgrade the `go` directive in `go.mod`.

## Features
- Custom router with params (`/users/:id`), wildcards (`/assets/*path`), groups, and host-based routing
- Method-not-allowed handling
- Named routes + path/query helpers
- Middleware chain (request ID, recovery, logging, CORS, body limit, timeout, auth, rate limiting)
- Security headers, IP allow/deny, CSRF protection
- Cookie-based session helper with signing
- Compression (gzip) + response ETag + cache control
- JSON and HTML rendering with layouts, template funcs, and reload in dev mode
- Static file helper with cache headers + ETag
- Metrics registry + JSON handler + histogram buckets
- Tracing hooks middleware
- Config defaults + env overrides + JSON config loader
- Validation helpers (including struct tags)
- Graceful shutdown helpers
- Minimal CLI generator + migration commands
- DB helpers + SQL migrations runner

## Quick Start (API)
```go
package main

import (
    "net/http"

    "github.com/devmarvs/bebo"
    "github.com/devmarvs/bebo/middleware"
)

func main() {
    app := bebo.New()
    app.Use(middleware.RequestID(), middleware.Recover(), middleware.Logger())

    app.GET("/health", func(ctx *bebo.Context) error {
        return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
    })

    _ = app.RunWithSignals()
}
```

## Routing & Groups
```go
api := app.Group("/api", middleware.RequestID())
v1 := api.Group("/v1")

v1.GET("/users/:id", handler)

// Or use version helper for /api/v1
app.Version("v1").GET("/health", handler)
```

## Host-Based Routing
```go
app.Route("GET", "/", handler, bebo.WithHost("example.com"))
app.Route("GET", "/", handler, bebo.WithHost("*.example.com"))
```

## Named Routes
```go
app.Route("GET", "/users/:id", handler, bebo.WithName("user.show"))
path, _ := app.Path("user.show", map[string]string{"id": "42"})
path, _ = app.PathWithQuery("user.show", map[string]string{"id": "42"}, map[string]string{"q": "test"})
```

## Static Assets
```go
app.Static("/static", "./public")
```

## Middleware Examples
```go
app.Use(
    middleware.CORS(middleware.CORSOptions{AllowedOrigins: []string{"https://example.com"}}),
    middleware.BodyLimit(2<<20),
    middleware.Timeout(5*time.Second),
    middleware.SecurityHeaders(middleware.DefaultSecurityHeaders()),
    middleware.Gzip(0),
)

limiter := middleware.NewLimiter(5, 10)
app.GET("/reports", reportsHandler, middleware.RateLimit(limiter))
```

## CSRF
```go
app.Use(middleware.CSRF(middleware.CSRFOptions{}))

app.POST("/submit", func(ctx *bebo.Context) error {
    token := middleware.CSRFToken(ctx)
    _ = token
    return ctx.Text(http.StatusOK, "ok")
})
```

## IP Allow/Deny
```go
filter, _ := middleware.IPFilter(middleware.IPFilterOptions{Allow: []string{"127.0.0.1"}})
app.Use(filter)
```

## Sessions
```go
store := session.NewCookieStore("bebo_session", []byte("secret"))

app.GET("/profile", func(ctx *bebo.Context) error {
    sess, _ := store.Get(ctx.Request)
    sess.Set("user_id", "123")
    _ = sess.Save(ctx.ResponseWriter)
    return ctx.Text(http.StatusOK, "ok")
})
```

## Metrics
```go
registry := metrics.New()
app.Use(middleware.Metrics(registry))

app.GET("/metrics", func(ctx *bebo.Context) error {
    metrics.Handler(registry).ServeHTTP(ctx.ResponseWriter, ctx.Request)
    return nil
})
```

## Auth Scaffolding
```go
type TokenAuth struct{}

func (a TokenAuth) Authenticate(ctx *bebo.Context) (*bebo.Principal, error) {
    token := ctx.Request.Header.Get("Authorization")
    if token == "" {
        return nil, nil
    }
    return &bebo.Principal{ID: "user-1"}, nil
}

app.GET("/private", privateHandler, middleware.RequireAuth(TokenAuth{}))
```

## Web Templating
Templates live in a directory (default `*.html`). If `LayoutTemplate` is set, each page template should `define "content"` and the layout should `template "content"`.

```go
resolver := assets.NewResolver("./public")
app := bebo.New(
    bebo.WithTemplateReload(true),
    bebo.WithTemplateFuncs(render.FuncMap{
        "asset": resolver.Func(),
    }),
)
```

See the runnable example:
```
examples/web
```

## Desktop (Fyne)
The desktop package is optional but included:
```
examples/desktop
```
Fyne is declared in `go.mod` for the desktop helpers.

## Configuration
Defaults are in `config.Default()` or `bebo.DefaultConfig()`. You can apply env overrides:
```go
cfg := bebo.ConfigFromEnv("BEBO_", bebo.DefaultConfig())
```
Load JSON config layered with env:
```go
cfg, err := bebo.LoadConfig("config.json", "BEBO_")
```
Env keys include: `ADDRESS`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `TEMPLATES_DIR`, `LAYOUT_TEMPLATE`, `TEMPLATE_RELOAD`.

## CLI
```sh
bebo new ./myapp -module github.com/me/myapp -template
bebo route add -method GET -path /users/:id -name user.show
bebo migrate new -dir ./migrations -name create_users
bebo migrate plan -dir ./migrations
```

## Migrations
```go
runner := migrate.New(db, "./migrations")
runner.Locker = migrate.AdvisoryLocker{ID: 42}
_, _ = runner.Up(context.Background())
```
Files use `0001_name.up.sql` and `0001_name.down.sql`.

## Layout
- `app.go`, `context.go`: core app and request context
- `router/`: custom router implementation
- `middleware/`: built-in middleware
- `render/`: JSON and HTML rendering
- `assets/`: cache-busting asset helper
- `config/`: config defaults + env overrides + JSON loader
- `validate/`: basic validation helpers
- `metrics/`: request metrics
- `session/`: cookie-backed sessions
- `db/`: database helpers
- `migrate/`: SQL migration runner
- `desktop/`: Fyne helpers
- `examples/`: API, web, and desktop samples

## Roadmap
See `ROADMAP.md`.
