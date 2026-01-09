# Production Runbook

## Pre-deploy checklist
- Configure BEBO_ environment overrides (timeouts, address, templates).
- Set session, JWT, and database secrets from a secret manager.
- Run migrations and verify the plan before deploying.

## Startup and shutdown
- Use app.RunWithSignals() for graceful shutdown.
- Configure BEBO_SHUTDOWN_TIMEOUT to match your load balancer drain window.

## Health and readiness
- Expose /health and /ready endpoints using health.Registry.
- Liveness should be fast and light; readiness should include DB/queue checks.

## Migrations
- Use bebo migrate plan/up in CI or a separate release step.
- Use advisory locks to avoid concurrent migrations.

## Observability
- Emit structured logs with request IDs and trace IDs.
- Export metrics and traces to your monitoring stack.
- Gate pprof endpoints behind auth and IP allowlists.

## Rollback
- Maintain down migrations or a rollback plan per release.
- Roll back app code first, then data when safe.

## Incident response
- Check /ready first, then logs, metrics, and pprof.
- Use rate limiting and feature flags to reduce blast radius.
