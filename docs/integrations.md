# Optional Integrations

Optional integrations live under `integrations/` as separate modules.

## Redis
Module: `github.com/devmarvs/bebo/integrations/redis`
- Redis client wrapper
- Cache/session/ratelimit helpers

## Postgres
Module: `github.com/devmarvs/bebo/integrations/postgres`
- pgx-based DB open helper
- Postgres session store convenience

## OpenTelemetry
Module: `github.com/devmarvs/bebo/integrations/otel`
- OpenTelemetry tracer adapter

These modules keep optional dependencies out of the core package.
