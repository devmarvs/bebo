# Roadmap

## v0.1 (current)
- Core app + custom router
- Route groups, named routes, and version helper
- Method-not-allowed handling
- Middleware (request ID, logging, recovery, CORS, body limit, timeout, auth scaffolding, rate limiting)
- Security headers + CSRF middleware
- Cookie-based session helper
- JSON and HTML rendering with layouts, template funcs, and reload
- Static file helper with cache headers + ETag
- Config defaults, env overrides, and JSON config loader
- Validation helpers with struct tags
- Metrics registry + JSON handler
- Graceful shutdown helpers
- DB helpers + SQL migration runner
- Minimal CLI generator
- Examples for API, web, desktop

## v0.2
- Static file fingerprint manifest integration
- Sessions store adapters
- Error reporting hooks and metrics helpers

## v0.3
- OpenAPI scaffolding
- CLI generator upgrades
- Optional integrations (caching, queues)
