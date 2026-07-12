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

Two env files: the repo root for the API and web, and `agent/` for the agent.

Root `.env` (API and web):

```bash
cp .env.example .env
```

The database defaults already match the local Postgres container. You need to set:

- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `NEXT_PUBLIC_GOOGLE_CLIENT_ID` — a Google OAuth 2.0 Web client with `http://localhost:3000` as an authorized origin (see [deployment.md](deployment.md))
- `JWT_SECRET` — generate with `openssl rand -hex 32`
- `USAGE_HASH_SECRET` — keys the usage ledger; generate with `openssl rand -hex 32`
- `RESEND_API_KEY` — optional locally; leave empty to use the fake mailer that logs magic links instead of sending
- `MAGIC_LINK_FROM` — verified Resend sender for sign-in links (only used when `RESEND_API_KEY` is set)
- `SIGNUP_OPEN=true` — accounts are closed by default; set this to create yours locally
- `LANGSMITH_API_KEY` — optional; when set (and tracing on in `agent/.env`), response feedback attaches to the run's LangSmith trace, otherwise it is DB-only

`AGENT_URL` defaults to `http://localhost:8081`, where `make agent-run` serves.

Agent `.env` (model and search keys):

```bash
cp agent/.env.example agent/.env
```

- `OPENROUTER_API_KEY` — chat completions ([openrouter.ai](https://openrouter.ai))
- `TAVILY_API_KEY` — web search ([tavily.com](https://tavily.com))
- `LANGSMITH_API_KEY` — optional run tracing ([smith.langchain.com](https://smith.langchain.com))

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
| **push** | Go tests, Vitest, pytest, `go build`, `tsc`, `terraform validate`, tflint |

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
