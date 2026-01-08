# Roadmap

## v0.1 (current)
- Core app + custom router
- Route groups, named routes, host-based routing, and version helper
- Method-not-allowed handling
- Middleware (request ID, logging, recovery, CORS, body limit, timeout, auth helpers, rate limiting)
- Security headers, IP allow/deny, CSRF middleware
- Security helpers (CSP builder, secure cookies, rotating JWT keys)
- Cookie-based session helper
- Redis cache adapter
- Compression (gzip), response ETag, cache control
- JSON and HTML rendering with layouts, template funcs, and reload
- Template partials + HTML error pages
- Static file helper with cache headers + ETag
- Config defaults, env overrides, and JSON config loader
- Config profiles (base/env/secrets) with validation
- Health/ready checks registry
- Validation helpers with struct tags
- OpenAPI builder + JSON handler
- HTTP client utilities (retry, backoff, circuit breaker)
- Task runner (in-process queue, retries/backoff, dead-letter hooks)
- Form/multipart binding + file upload helpers
- Configurable logging/metrics/tracing middleware options
- Rate limiting policies + Redis token bucket adapter
- JWT authenticator helper
- Session store adapters (memory, redis, postgres)
- Metrics registry + JSON handler + latency buckets
- Tracing hooks middleware
- Observability: structured request logs + auth-gated pprof endpoints
- Prometheus exporter + optional OpenTelemetry adapter
- Realtime helpers (SSE/WebSocket)
- Graceful shutdown helpers
- DB helpers + SQL migration runner with plan
- DB query helpers with timeouts + migration lock improvements
- Minimal CLI generator + migration commands
- Examples for API, web, desktop

## v0.2
- Static file fingerprint manifest integration
- Error reporting hooks and metrics helpers
- Brotli compression (third-party)

## v0.3
- CLI generator upgrades
- Optional integrations (caching, queues)
