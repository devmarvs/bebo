# Secrets Handling Checklist

- Keep secrets out of source control. Never commit .env, keys, or DSNs.
- Prefer a secret manager (KMS, Vault, SSM) over plain environment files.
- Use distinct secrets per environment and rotate them regularly.
- Store secrets in the smallest scope possible (app-only, least privilege).
- Redact secrets from logs, errors, and tracing payloads.
- Treat cookies, Authorization headers, and session IDs as secrets.
- For local dev, use short-lived keys and do not reuse production secrets.
- Use config profiles (base + env + secrets) and keep secrets outside the repo.
