# bebo

bebo is a batteries-included Go framework focused on building REST APIs and server-rendered web apps with a lightweight custom router. Desktop support is available via Fyne.

## Status
- v0.1 scaffold with custom router, middleware, JSON/HTML rendering, config defaults, and examples.
- Desktop helpers live in `desktop/` and depend on Fyne.

## Requirements
- Go 1.25 (as requested). If you are on a released Go toolchain, downgrade the `go` directive in `go.mod`.

## Features
- Custom router with params (`/users/:id`) and wildcards (`/assets/*path`)
- Middleware chain (request ID, recovery, logging)
- JSON and HTML rendering
- Config defaults + env overrides
- Validation helpers

## Quick Start (API)
```go
package main

import (
    "log"
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

    if err := app.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

## Web Templating
Templates live in a directory (default `*.html`). If `LayoutTemplate` is set, each page template should `define "content"` and the layout should `template "content"`.

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

## Layout
- `app.go`, `context.go`: core app and request context
- `router/`: custom router implementation
- `middleware/`: built-in middleware
- `render/`: JSON and HTML rendering
- `config/`: config defaults + env overrides
- `validate/`: basic validation helpers
- `desktop/`: Fyne helpers
- `examples/`: API, web, and desktop samples

## Roadmap
See `ROADMAP.md`.
