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
| Agent errors | more than 2 agent error logs in 5 minutes |
| Agent 5xx | more than 5 agent 5xx responses in 5 minutes |
| LB 429 spike | more than 30 rate-limited responses in 5 minutes |

When an alert fires, the [runbooks](runbooks/) say what to do.

All checks and policies are defined in [infra/monitoring.tf](../infra/monitoring.tf) and route to the owner email channel. View their live status in the Cloud Monitoring console.

## Billing budget

A monthly budget (default $20) emails the owner as spend crosses 50%, 90%, 100%, and 150%, plus a forecast warning, so a cost spike surfaces early. It is notify-only; [full-kill](runbooks/full-kill.md) is the response. Defined in [infra/budget.tf](../infra/budget.tf).

## Tracing

The agent traces every run to LangSmith for step-level inspection of reasoning, tool calls, and subagents beyond what the Cloud Run logs show.

LangSmith online evaluators score live prod traces (see [deployment.md](deployment.md#langsmith-online-evals)), and users rate replies with a thumbs up/down (see [agent/evals/README.md](../agent/evals/README.md#user-feedback)). Three alerts email the owner through Resend:

| Alert | Fires when | Signal |
| --- | --- | --- |
| PII in answers | a prod answer scores positive on the pii evaluator | security |
| Prompt injection | a user message scores positive on the prompt-injection judge | security |
| Negative feedback spike | 5 or more replies get a thumbs-down in an hour | quality |

The security alerts trip on any single positive over a 15 minute window; the feedback alert needs 5 within a 60 minute window. Defined in [infra/langsmith.tf](../infra/langsmith.tf). When one fires, [triage the trace](runbooks/eval-alerts.md).

The offline eval suite also runs weekly and emails the owner its results; see [agent/evals/README.md](../agent/evals/README.md#running).
