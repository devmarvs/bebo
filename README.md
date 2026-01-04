# bebo

bebo is a batteries-included Go framework focused on building REST APIs and server-rendered web apps with a lightweight custom router. Desktop support is available via Fyne.

## Status
- v0.1 scaffold with custom router, middleware, JSON/HTML rendering, config defaults, static assets, and examples.
- Desktop helpers live in `desktop/` and depend on Fyne.

## Requirements
- Go 1.25 (as requested). If you are on a released Go toolchain, downgrade the `go` directive in `go.mod`.

## Features
- Custom router with params (`/users/:id`), wildcards (`/assets/*path`), and groups
- Middleware chain (request ID, recovery, logging, CORS, body limit, timeout, auth, rate limiting)
- JSON and HTML rendering with layouts, template funcs, and reload in dev mode
- Static file helper with cache headers + ETag
- Config defaults + env overrides
- Validation helpers
- Graceful shutdown helpers

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
)

limiter := middleware.NewLimiter(5, 10)
app.GET("/reports", reportsHandler, middleware.RateLimit(limiter))
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
Env keys include: `ADDRESS`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `TEMPLATES_DIR`, `LAYOUT_TEMPLATE`, `TEMPLATE_RELOAD`.

## Layout
- `app.go`, `context.go`: core app and request context
- `router/`: custom router implementation
- `middleware/`: built-in middleware
- `render/`: JSON and HTML rendering
- `assets/`: cache-busting asset helper
- `config/`: config defaults + env overrides
- `validate/`: basic validation helpers
- `desktop/`: Fyne helpers
- `examples/`: API, web, and desktop samples

## Roadmap
See `ROADMAP.md`.
