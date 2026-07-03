# Monitoring

Cloud Monitoring watches uptime, errors, latency, and the database. Every alert notifies the owner by email.

## Uptime checks

Two HTTPS checks run every 60 seconds:

- `/readyz` on the API, which also verifies the database is reachable
- the web root, which confirms the frontend is serving

## Alerts

| Alert | Fires when |
| --- | --- |
| Site down | the `/readyz` check fails |
| Web down | the web root check fails |
| API 5xx | more than 5 server errors in 5 minutes |
| API latency | non-stream p95 latency exceeds 2s over 5 minutes |
| Cloud SQL disk | disk utilization exceeds 85% for 10 minutes |
| Cloud SQL CPU | CPU utilization exceeds 80% for 15 minutes |
| OpenRouter errors | more than 2 upstream errors in 5 minutes |
| LB 429 spike | more than 30 rate-limited responses in 5 minutes |

All checks and policies are defined in [infra/monitoring.tf](../infra/monitoring.tf) and route to the owner email channel. View their live status in the Cloud Monitoring console.
