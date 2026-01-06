# Roadmap

## v0.1 (current)
- Core app + custom router
- Route groups, named routes, host-based routing, and version helper
- Method-not-allowed handling
- Middleware (request ID, logging, recovery, CORS, body limit, timeout, auth scaffolding, rate limiting)
- Security headers, IP allow/deny, CSRF middleware
- Cookie-based session helper
- Compression (gzip), response ETag, cache control
- JSON and HTML rendering with layouts, template funcs, and reload
- Template partials + HTML error pages
- Static file helper with cache headers + ETag
- Config defaults, env overrides, and JSON config loader
- Validation helpers with struct tags
- OpenAPI scaffolding (builder + JSON handler)
- HTTP client utilities (retry, backoff, circuit breaker)
- Form/multipart binding + file upload helpers
- Configurable logging/metrics/tracing middleware options
- JWT authenticator helper
- Session store adapters (memory)
- Metrics registry + JSON handler + latency buckets
- Tracing hooks middleware
- Prometheus exporter + optional OpenTelemetry adapter
- Graceful shutdown helpers
- DB helpers + SQL migration runner with plan
- Minimal CLI generator + migration commands
- Examples for API, web, desktop

## v0.2
- Static file fingerprint manifest integration
- Session store adapters (redis, postgres)
- Error reporting hooks and metrics helpers
- Brotli compression (third-party)

## v0.3
- CLI generator upgrades
- Optional integrations (caching, queues)
