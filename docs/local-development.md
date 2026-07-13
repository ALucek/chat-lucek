# Local Development

How to run chat-lucek on your machine and the workflow for changing it.

## Prerequisites

- **Go** 1.26+
- **Node** 20+ and **pnpm**
- **uv** ([Astral](https://docs.astral.sh/uv/)) runs the Python agent
- **Docker** (runs local Postgres)
- **pre-commit** (`brew install pre-commit`) for the git hooks
- **gitleaks** (secret-scanning commit hook)
- **shellcheck** (`brew install shellcheck`) so actionlint lints workflow `run:` scripts as CI does
- **Terraform** 1.9+ and **tflint** (infra `.tf` hooks)

## Environment

Two env files. The repo root holds everything the API and web need; `agent/` holds the agent's own keys. The agent stays separate because it runs as its own service and never needs the database or auth secrets.

```bash
cp .env.example .env
cp agent/.env.example agent/.env
```

The `make` targets load the root `.env` for you, including `make web-run`, so the web dev server reads it too. There is no separate `web/.env.local`. Database defaults already match the local Postgres container.

To sign in locally: set the three Google vars, generate the two secrets, and set `SIGNUP_OPEN=true` to create your account.

### Root `.env`

| Variable | Required | Purpose |
| --- | --- | --- |
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` | yes | Local Postgres; defaults match the container |
| `DATABASE_URL` | no | Full DSN; overrides the `DB_*` parts when set |
| `PORT` | yes | API port (default 8080) |
| `ALLOWED_ORIGIN` | no | CORS origin (default `http://localhost:3000`) |
| `AGENT_URL` | no | Agent base URL (default `http://localhost:8081`, where `make agent-run` serves) |
| `LOG_LEVEL` | no | debug, info, warn, error (default info) |
| `RUNS_BUDGET_DAILY` | no | Per-user rolling-24h run cap (default 20) |
| `JWT_SECRET` | yes | Signs session tokens; `openssl rand -hex 32` |
| `USAGE_HASH_SECRET` | yes | Keys the usage ledger; `openssl rand -hex 32` |
| `GOOGLE_CLIENT_ID` `GOOGLE_CLIENT_SECRET` | yes | Google OAuth 2.0 web client with `http://localhost:3000` as an authorized origin (see [deployment.md](deployment.md)) |
| `OWNER_EMAIL` | no | This email gets an unlimited daily run budget |
| `SIGNUP_OPEN` | no | `true` to allow new accounts (closed by default) |
| `RESEND_API_KEY` | no | Magic-link email; empty uses the fake mailer that logs links |
| `MAGIC_LINK_FROM` | no | Verified Resend sender; required when `RESEND_API_KEY` is set |
| `NEXT_PUBLIC_API_URL` | yes | Where the browser calls the API; unset means same-origin. Local dev uses `http://localhost:8080` |
| `NEXT_PUBLIC_GOOGLE_CLIENT_ID` | yes | Same value as `GOOGLE_CLIENT_ID`, exposed to the browser |
| `LANGSMITH_API_KEY` | no | Attaches response feedback to the run's LangSmith trace; empty keeps it DB-only |
| `LANGSMITH_ENDPOINT` | no | LangSmith API base URL (default `https://api.smith.langchain.com`) |

### Agent `.env`

| Variable | Required | Purpose |
| --- | --- | --- |
| `OPENROUTER_API_KEY` | yes | Chat completions ([openrouter.ai](https://openrouter.ai)) |
| `TAVILY_API_KEY` | yes | Web search ([tavily.com](https://tavily.com)) |
| `LANGSMITH_API_KEY` | no | Run tracing ([smith.langchain.com](https://smith.langchain.com)); same value as the root `.env` key |
| `LANGSMITH_TRACING` | no | `true` to send traces |
| `LANGSMITH_ENDPOINT` `LANGSMITH_PROJECT` | no | Tracing destination |
| `OPENAI_API_KEY` | no | Only to publish online LLM-judge evaluators; any placeholder works |
| `DEFAULT_MODEL` | no | Agent model (default `deepseek/deepseek-v4-flash`) |
| `MAX_SEARCHES` | no | Web searches per agent run (default 5) |
| `MAX_TOKENS` | no | Max output tokens per model call (default 8192) |
| `RECURSION_LIMIT` `SUBAGENT_RECURSION_LIMIT` | no | LangGraph recursion caps (defaults 100 and 50) |
| `MODEL_MAX_RETRIES` | no | Retries per model call (default 3) |

## Run the stack

Two ways to bring it up locally: run each service directly (each in its own terminal) for fast reloads, or run the whole thing in containers.

| Command | Does |
| --- | --- |
| `make db-up` | Start Postgres in Docker |
| `make migrate-up` | Apply database migrations |
| `make agent-run` | Run the agent on :8081 |
| `make api-run` | Run the API on :8080 |
| `make web-run` | Run the web app on :3000 |
| `make stack-up` | Build and run web, api, agent, and db in containers |
| `make stack-down` | Stop the container stack |

For direct runs, start `db-up` and `migrate-up` first, then the three services, then open <http://localhost:3000>. The containerized agent reads its keys from `agent/.env`.

## Git hooks

Install the hooks once per clone:

```bash
make hooks
```

They run in two stages so commits stay fast and the slow checks gate the push:

| Stage | Checks |
| --- | --- |
| **commit** | gofmt, go vet, Prettier, ESLint, Ruff, comment style, gitleaks, `terraform fmt`, actionlint |
| **push** | Go tests, Vitest, pytest, `go build`, `tsc`, migration lint, `terraform validate`, tflint |

## Everyday commands

| Command | Does |
| --- | --- |
| `make fmt` | Format Go, web, and agent code |
| `make lint` | Format-check, vet, ESLint, Ruff, comment style |
| `make typecheck` | `go build` and `tsc` |
| `make test` | Go, web, and agent unit tests |
| `make api-check` / `web-check` / `agent-check` | One service's full gate (what CI runs for it) |
| `make e2e-local` | Start the DB, run the Playwright e2e suite, tear down |
| `make security` | Static security scans (govulncheck, gosec, pnpm audit, pip-audit, bandit, gitleaks) |
| `make check` | Full pre-merge gate (per-service gates + infra and workflow checks + e2e) |
| `make evals` | Run the agent's live behavioral evals (real models; see [testing.md](testing.md)) |
| `make db-reset` | Wipe, recreate, and migrate the local database |
| `make db-psql` | Open a `psql` shell into the local database |
| `make health` | Curl the API's `/readyz` |

For the test suites in depth, see [testing.md](testing.md).
