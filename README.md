# bebo

bebo is a batteries-included Go framework focused on building REST APIs and server-rendered web apps with a lightweight custom router. Desktop support is available via Fyne.

## Status
- v0.1 release with custom router, middleware, JSON/HTML rendering, config defaults, static assets, and examples.
- Desktop helpers live in `desktop/` and depend on Fyne.

## Requirements
- Go 1.25 (as requested). If you are on a released Go toolchain, downgrade the `go` directive in `go.mod`.

## Features
- Custom router with params (`/users/:id`), wildcards (`/assets/*path`), groups, and host-based routing
- Method-not-allowed handling
- Named routes + path/query helpers
- Middleware chain (request ID, recovery, logging, CORS, body limit, timeout, auth, rate limiting)
- Rate limiting (in-memory + Redis token bucket) with per-route policies
- Security headers, IP allow/deny, CSRF protection
- Security helpers: CSP builder, secure cookies, rotating JWT keys
- Cookie-based sessions + memory/redis/postgres stores
- Redis cache adapter
- Flash messages (session-backed) + CSRF template helpers
- Method override for HTML forms (PUT/PATCH/DELETE)
- Compression (gzip) + response ETag + cache control
- JSON and HTML rendering with layouts, template funcs, partials, reload, and embedded templates
- HTML error pages with configurable templates
- Health/ready checks registry
- Static file helper with cache headers + ETag (disk or fs.FS)
- Metrics registry + JSON handler + histogram buckets + Prometheus exporter
- Tracing hooks middleware + optional OpenTelemetry adapter
- Observability: structured access logs (trace/span IDs, request/response bytes) + auth-gated pprof endpoints
- Request metadata propagation helpers (traceparent/request IDs)
- Realtime (SSE/WebSocket) helpers
- OpenAPI builder + JSON handler
- HTTP client utilities (timeouts, retries, backoff, circuit breaker)
- Background job runner (in-process queue, retries/backoff, dead-letter hooks)
- Form/multipart binding + file upload helpers
- Config defaults + env overrides + JSON config loader + layered profiles (base/env/secrets) with validation
- Validation helpers (including struct tags, non-string fields, and custom validators)
- Extensibility registry + auth/cache/validation hooks
- Optional integrations as submodules (redis/postgres/otel)
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

## OpenAPI
```go
spec := openapi.New(openapi.Info{Title: "bebo app", Version: "v0.1"})
_ = app.AddOpenAPIRoutes(spec, bebo.WithOpenAPIIncludeUnnamed(false))

app.GET("/openapi.json", func(ctx *bebo.Context) error {
    openapi.Handler(spec.Document()).ServeHTTP(ctx.ResponseWriter, ctx.Request)
    return nil
})
```

## Static Assets
```go
app.Static("/static", "./public")
app.StaticFS("/static", staticFS)
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

## Middleware Options
```go
logOpts := middleware.DefaultLoggerOptions()
logOpts.SkipPaths = []string{"/health"}
app.Use(middleware.LoggerWithOptions(logOpts))

metricsOpts := middleware.DefaultMetricsOptions(registry)
metricsOpts.SkipPaths = []string{"/metrics"}
app.Use(middleware.MetricsWithOptions(metricsOpts))

traceOpts := middleware.DefaultTraceOptions(tracer)
traceOpts.SkipPaths = []string{"/metrics"}
app.Use(middleware.TraceWithOptions(traceOpts))
```

## Rate Limiting (Redis + Policies)
```go
redisLimiter := middleware.NewRedisLimiter(5, 10, middleware.RedisLimiterOptions{Address: "127.0.0.1:6379"})
policies, _ := middleware.NewRateLimitPolicies([]middleware.RateLimitPolicy{
    {
        Method: http.MethodGet,
        Path:   "/reports/:id",
        Allow: func(ctx *bebo.Context, key string) (bool, error) {
            return redisLimiter.Allow(ctx.Request.Context(), key)
        },
    },
})
app.Use(policies.Middleware())
```

## Request Metadata Propagation
```go
app.Use(middleware.RequestID(), middleware.RequestContext())

runner := tasks.New(tasks.DefaultOptions())
runner.Start(context.Background())

app.POST("/jobs", func(ctx *bebo.Context) error {
    runner.Enqueue(tasks.Job{
        Context: ctx.Request.Context(),
        Handler: func(jobCtx context.Context) error {
            logger := bebo.LoggerFromContext(jobCtx, slog.Default())
            logger.Info("job started")
            return nil
        },
    })
    return ctx.Text(http.StatusAccepted, "queued")
})
```
Use `bebo.InjectRequestMetadata` to copy headers to outgoing HTTP requests.

## CSRF
```go
app.Use(middleware.CSRF(middleware.CSRFOptions{}))

app.POST("/submit", func(ctx *bebo.Context) error {
    token := middleware.CSRFToken(ctx)
    _ = token
    return ctx.Text(http.StatusOK, "ok")
})
```

## CSP Builder
```go
policy := security.NewCSP().
    DefaultSrc("'self'").
    ScriptSrc("'self'", "cdn.example.com").
    UpgradeInsecureRequests()
app.Use(middleware.SecurityHeaders(middleware.SecurityHeadersOptions{
    ContentSecurityPolicy: policy.String(),
}))
```

## Secure Cookies
```go
cookie := security.NewSecureCookie("session", "value", security.CookieOptions{})
http.SetCookie(ctx.ResponseWriter, cookie)
```

## Method Override (HTML forms)
```go
app.UsePre(middleware.MethodOverride(middleware.MethodOverrideOptions{}))
```
Forms can send a hidden `_method` field to trigger PUT/PATCH/DELETE.


## IP Allow/Deny
```go
filter, _ := middleware.IPFilter(middleware.IPFilterOptions{Allow: []string{"127.0.0.1"}})
app.Use(filter)
```

## Sessions
```go
cookieStore := session.NewCookieStore("bebo_session", []byte("secret"))
memoryStore := session.NewMemoryStore("bebo_session", 30*time.Minute)

app.GET("/profile", func(ctx *bebo.Context) error {
    sess, _ := cookieStore.Get(ctx.Request)
    sess.Set("user_id", "123")
    _ = sess.Save(ctx.ResponseWriter)
    return ctx.Text(http.StatusOK, "ok")
})
```

## Persistent Sessions (Redis/Postgres)
```go
redisStore := session.NewRedisStore(session.RedisOptions{
    Address: "127.0.0.1:6379",
    TTL:     30 * time.Minute,
})

pgStore, _ := session.NewPostgresStore(session.PostgresOptions{
    DB:   db,
    TTL:  30 * time.Minute,
    Table: "bebo_sessions",
})
_ = pgStore.EnsureTable(context.Background())
```

Note: Postgres requires a driver (pgx/pq) in your app.

## Cache (Redis)
```go
store := cache.NewRedisStore(cache.RedisOptions{
    Address:    "127.0.0.1:6379",
    DefaultTTL: 5 * time.Minute,
})
_ = store.Set(context.Background(), "user:1", []byte("cached"), 0)
```

## Metrics
```go
registry := metrics.New()
app.Use(middleware.Metrics(registry))

app.GET("/metrics", func(ctx *bebo.Context) error {
    metrics.PrometheusHandler(registry).ServeHTTP(ctx.ResponseWriter, ctx.Request)
    return nil
})
```
Use `metrics.Handler(registry)` for JSON snapshots.

## Pprof (Authenticated)
```go
authenticator := auth.JWTAuthenticator{Key: []byte("secret")}
_ = pprof.Register(app, authenticator)
```

## Health & Readiness
```go
registry := health.New(health.WithTimeout(500 * time.Millisecond))
registry.Add("db", func(ctx context.Context) error {
    return db.PingContext(ctx)
})
registry.AddReady("cache", func(ctx context.Context) error {
    return cache.Ping(ctx)
})

app.GET("/healthz", func(ctx *bebo.Context) error {
    registry.Handler().ServeHTTP(ctx.ResponseWriter, ctx.Request)
    return nil
})
app.GET("/readyz", func(ctx *bebo.Context) error {
    registry.ReadyHandler().ServeHTTP(ctx.ResponseWriter, ctx.Request)
    return nil
})
```

## HTTP Client
```go
breaker := httpclient.NewCircuitBreaker(httpclient.CircuitBreakerOptions{
    MaxFailures:  5,
    ResetTimeout: 30 * time.Second,
})

client := httpclient.NewClient(httpclient.ClientOptions{
    Timeout: 10 * time.Second,
    Retry:   httpclient.DefaultRetryOptions(),
    Breaker: breaker,
})
```


## Background Jobs
```go
runner := tasks.New(tasks.DefaultOptions())
runner.Start(context.Background())

_ = runner.Enqueue(tasks.Job{
    Name:    "sync-users",
    Context: context.Background(),
    Handler: func(ctx context.Context) error {
        return nil
    },
})
```

## Realtime (SSE/WebSocket)
```go
app.GET("/events", func(ctx *bebo.Context) error {
    stream, err := realtime.StartSSE(ctx, realtime.SSEOptions{})
    if err != nil {
        return err
    }
    return stream.Send(realtime.SSEMessage{Event: "status", Data: "ok"})
})

app.GET("/ws", func(ctx *bebo.Context) error {
    conn, err := realtime.Upgrade(ctx, realtime.WebSocketOptions{})
    if err != nil {
        return err
    }
    defer conn.Close()

    for {
        msg, err := conn.ReadText()
        if err != nil {
            return err
        }
        _ = conn.WriteText("echo: " + msg)
    }
})
```

## OpenTelemetry (optional)
OpenTelemetry support lives behind the `otel` build tag. Add the OpenTelemetry SDK to your project and build with `-tags otel`.

```go
tracer, _ := otel.NewTracer("bebo")
app.Use(middleware.Trace(tracer))
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

## JWT Auth
```go
authenticator := auth.JWTAuthenticator{
    Key:      []byte("secret"),
    Issuer:   "bebo",
    Audience: "api",
}

app.GET("/private", privateHandler, middleware.RequireAuth(authenticator))
```

Rotating keys:
```go
keys := auth.JWTKeySet{
    Primary: auth.JWTKey{ID: "v2", Secret: []byte("new")},
    Fallback: []auth.JWTKey{
        {ID: "v1", Secret: []byte("old")},
    },
}

// Use keys.Sign to mint tokens with the primary key.
_ = keys

rotating := auth.JWTAuthenticator{KeySet: &keys}
app.GET("/private", privateHandler, middleware.RequireAuth(rotating))
```

## Request Binding
```go
type Signup struct {
    Email string `form:"email"`
    Age   int    `form:"age"`
}

var form Signup
if err := ctx.BindForm(&form); err != nil {
    return err
}

file, _ := ctx.FormFile("avatar", bebo.DefaultMultipartMemory)
_ = ctx.SaveUploadedFile(file, "/tmp/"+file.Filename)
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

## Template Helpers (CSRF + Flash)
```go
store := flash.New(session.NewCookieStore("bebo_session", []byte("secret")))

app := bebo.New(
    bebo.WithTemplateFuncs(web.Funcs()),
)
app.Use(middleware.CSRF(middleware.CSRFOptions{}))

app.GET("/", func(ctx *bebo.Context) error {
    view, err := web.TemplateDataFrom(ctx, &store, pageData)
    if err != nil {
        return err
    }
    return ctx.HTML(http.StatusOK, "home.html", view)
})
```
Template usage:
```html
<form method="post">
  {{ csrfField .CSRFToken }}
</form>
<ul>
  {{ range .Flash }}
  <li class="flash-{{ .Type }}">{{ .Text }}</li>
  {{ end }}
</ul>
```

## Template Partials
Enable nested templates and partial discovery. By default, files under `partials/` or prefixed with `_` are treated as partials when subdir loading is enabled.

```go
app := bebo.New(
    bebo.WithTemplateSubdirs(true),
    bebo.WithTemplatePartials("partials/**/*.html", "_*.html"),
)
```

## Embedded Templates & Assets
```go
import (
    "embed"
    "io/fs"
)

//go:embed templates/* static/*
var embedded embed.FS

tmplFS, _ := fs.Sub(embedded, "templates")
staticFS, _ := fs.Sub(embedded, "static")

app := bebo.New(
    bebo.WithTemplateFS(tmplFS, "."),
    bebo.WithTemplateFSDevDir("templates"),
    bebo.WithTemplateReload(true),
)
app.StaticFS("/static", staticFS)
```

## HTML Error Pages
If your error templates live in nested directories, enable `bebo.WithTemplateSubdirs(true)`. Error templates receive `ErrorPageData` with a nested `Error` envelope and `RequestID`.
```go
app := bebo.New(
    bebo.WithRenderer(engine),
    bebo.WithErrorTemplates(map[int]string{
        http.StatusNotFound:           "errors/404.html",
        http.StatusInternalServerError: "errors/500.html",
        0:                             "errors/default.html",
    }),
)
```

## Desktop (Fyne)
The desktop package is optional but included:
```
examples/desktop
```
Helpers cover window icons, menus, and tray menus via `desktop.WindowConfig`.
To package a desktop app with an icon:
```sh
fyne package -os darwin -icon path/to/icon.png
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
Load layered profiles (base + env + secrets) with validation:
```go
profile := bebo.ConfigProfile{
    BasePath:    "config/base.json",
    EnvPath:     "config/production.json",
    SecretsPath: "config/secrets.json",
    EnvPrefix:   "BEBO_",
}
cfg, err := bebo.LoadConfigProfile(profile)
```
Typed loaders are available via `config.Loader[T]` for custom config structs.
Env keys include: `ADDRESS`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `TEMPLATES_DIR`, `LAYOUT_TEMPLATE`, `TEMPLATE_RELOAD`.

## Versioning
See `VERSIONING.md`, `DEPRECATION.md`, and `CHANGELOG.md`.

## Docs
- Security: `SECURITY.md`
- Hardening: `docs/hardening.md`
- Authentication defaults: `docs/authentication.md`
- Secrets checklist: `docs/secrets.md`
- Crypto/key rotation: `docs/crypto-keys.md`
- Production runbook: `docs/runbook.md`
- Scaling: `docs/scaling.md`
- Timeouts & circuit breakers: `docs/timeouts.md`
- Config profiles: `docs/config-profiles.md`
- Extensibility: `docs/extensibility.md`
- Integrations: `docs/integrations.md`
- Project structure: `docs/project-structure.md`
- Scaffolding upgrades: `docs/scaffolding-upgrades.md`
- Migration guide: `docs/migration-guide.md`
- Deployment examples: `deploy/docker/Dockerfile`, `deploy/k8s/`
- CRUD app example: `examples/crud`

## CLI
```sh
bebo new ./myapp -module github.com/me/myapp -web -template -profile
bebo route add -method GET -path /users/:id -name user.show
bebo crud new users -dir handlers -package handlers -templates templates
bebo migrate new -dir ./migrations -name create_users
bebo migrate plan -dir ./migrations
```
Supports `-api`, `-web`, and `-desktop` scaffolds.


## DB Helpers
```go
helper := db.Helper{Timeout: 2 * time.Second}
_, _ = helper.Exec(context.Background(), dbConn, "SELECT 1")

repo := db.NewRepository(dbConn, 2*time.Second)
limit, offset := db.Pagination{Page: 1, Size: 25}.LimitOffset()
query, args, _ := db.Select("id", "name").From("users").Where("active = ?", true).Build()
_ = limit
_ = offset
_ = query
_ = args

logged := db.WithQueryHook(dbConn, func(ctx context.Context, q string, args []any, d time.Duration, err error) {})
_ = logged
```

## Migrations
```go
runner := migrate.New(db, "./migrations")
runner.Locker = migrate.AdvisoryLocker{ID: 42, Timeout: 5 * time.Second}
_, _ = runner.Up(context.Background())
```
Files use `0001_name.up.sql` and `0001_name.down.sql`.

## Layout
- `app.go`, `context.go`: core app and request context
- `router/`: custom router implementation
- `middleware/`: built-in middleware
- `render/`: JSON and HTML rendering
- `assets/`: cache-busting asset helper
- `compat/`: backwards-compatibility tests
- `cache/`: Redis cache adapter
- `redis/`: shared Redis client
- `security/`: CSP builder + secure cookies
- `web/`: template helpers (csrf + flash)
- `config/`: config defaults + env overrides + JSON loader + profiles
- `validate/`: basic validation helpers
- `metrics/`: request metrics + Prometheus exporter
- `pprof/`: authenticated pprof endpoints
- `health/`: health and readiness checks
- `session/`: cookie-backed sessions + memory store
- `flash/`: flash messages
- `auth/`: JWT authenticator helper
- `openapi/`: OpenAPI builder + handler
- `otel/`: OpenTelemetry adapter (build tag)
- `httpclient/`: HTTP client utilities (retry/backoff/breaker)
- `tasks/`: background jobs runner
- `realtime/`: SSE + WebSocket helpers
- `db/`: database helpers
- `migrate/`: SQL migration runner
- `desktop/`: Fyne helpers
- `docs/`: security, runbooks, and guides
- `deploy/`: container and Kubernetes examples
- `integrations/`: optional submodules (redis/postgres/otel)
- `examples/`: API, web, desktop, and CRUD samples

## Roadmap
See `ROADMAP.md`.
