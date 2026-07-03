# Runbooks

What to do to operate and recover chat-lucek. Most levers are one-click GitHub Actions workflows (run from the Actions tab); a few are Terraform or gcloud. When an [alert](../monitoring.md) fires, start here.

## When an alert fires

| Alert | Response |
| --- | --- |
| Site down (readyz) | Recent deploy? [Roll it back](rollback.md). Otherwise check the API logs and that the database is reachable ([restore](db-restore.md) if it is not). |
| Web down (root) | Bad `chat-web` revision. [Roll web back](rollback.md). |
| API 5xx | Almost always the last deploy. [Roll back](rollback.md). |
| API latency | Check Cloud SQL CPU and the API logs; suspect a slow query, cold starts, or a slow upstream. |
| Cloud SQL disk >85% | Autoresize should cover it; if growth is runaway, investigate and raise the disk. |
| Cloud SQL CPU >80% | Check for slow queries or a connection storm; consider a larger tier. |
| OpenRouter errors | Provider down, key dead, or out of credits. Check OpenRouter status and the key. |
| LB 429 spike | Abuse. Climb the [escalation ladder](abuse.md). |

## Runbooks

- [Rollback](rollback.md) — shift traffic to a prior revision
- [Maintenance mode](maintenance.md) — serve a maintenance page and 503 the API
- [Rotate JWT](rotate-jwt.md) — rotate the signing key, invalidating live tokens
- [Rotate secrets](rotate-secrets.md) — rotate the database, OpenRouter, and Google secrets
- [Disable signup](disable-signup.md) — open or close registration
- [Full-kill](full-kill.md) — deny all API traffic at the edge
- [Restore the database](db-restore.md) — recover Cloud SQL from a backup
- [Abuse response](abuse.md) — escalation ladder for a traffic or abuse spike
