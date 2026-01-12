# CRUD Example Runbook

This runbook covers production operations for the CRUD example (auth + sessions + migrations).

## Configuration
- `BEBO_DATABASE_URL`: Postgres DSN (required).
- `BEBO_SESSION_KEY`: 32+ byte random key for session cookies.
- `BEBO_SECURE_COOKIES`: set to `true` behind HTTPS.
- `BEBO_AUTO_MIGRATE`: set to `true` only in controlled environments.

## Migrations
- Prefer running `go run ./examples/crud -migrate` in CI/CD before deploy.
- Use advisory locks for serialized migrations (already configured in the example).

## Health and readiness
- `/health` is liveness and should be fast.
- `/ready` checks DB connectivity; wire it into load balancer readiness checks.

## Sessions and auth
- Rotate session signing keys on a schedule.
- Use HTTPS in production and enable `BEBO_SECURE_COOKIES=true`.

## Observability
- Enable request IDs, structured logs, and metrics.
- Export `/metrics` and `/pprof` behind auth and IP allowlists.

## Scaling
- Use a connection pool sized to your DB limits.
- Use Redis/Postgres session stores for multiple instances.

## Backup and recovery
- Back up the database regularly.
- Test restoring from backups in a staging environment.

## Rollback
- Maintain down migrations for each release.
- Roll back app code first, then data if required.
