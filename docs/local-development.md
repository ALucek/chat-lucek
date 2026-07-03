# Local Development

How to run chat-lucek on your machine and the workflow for changing it.

## Prerequisites

- **Go** 1.26+
- **Node** 20+ and **pnpm**
- **Docker** (runs local Postgres)
- **pre-commit** (`brew install pre-commit`) for the git hooks
- **gitleaks** (secret-scanning commit hook)
- **Terraform** 1.9+ and **tflint** (infra `.tf` hooks)

## Environment

Copy the example env file and fill in the secrets. Run from the repo root:

```bash
cp .env.example .env
```

The database defaults already match the local Postgres container. You need to set:

- `OPENROUTER_API_KEY` — for chat completions ([openrouter.ai](https://openrouter.ai))
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `NEXT_PUBLIC_GOOGLE_CLIENT_ID` — a Google OAuth 2.0 Web client with `http://localhost:3000` as an authorized origin (see [deployment.md](deployment.md))
- `JWT_SECRET` — generate with `openssl rand -hex 32`
- `SIGNUP_OPEN=true` — accounts are closed by default; set this to create yours locally

## Run the stack

Two ways. For day-to-day work, run each service directly so you get fast reloads:

```bash
make db-up          # Postgres in Docker
make migrate-up     # apply migrations
make api-run        # API on :8080  (separate terminal)
make web-run        # web on :3000  (separate terminal)
```

Open <http://localhost:3000>.

To run everything in containers instead:

```bash
make stack-up       # build and run web, api, and db
make stack-down
```

## Git hooks

Install the hooks once per clone:

```bash
make hooks
```

They run in two stages so commits stay fast and the slow checks gate the push:

| Stage | Checks |
| --- | --- |
| **commit** | gofmt, go vet, Prettier, ESLint, comment style, gitleaks, `terraform fmt` |
| **push** | Go tests, Vitest, `go build`, `tsc`, `terraform validate`, tflint |

## Everyday commands

| Command | Does |
| --- | --- |
| `make fmt` | Format Go and web code |
| `make lint` | Format-check, vet, ESLint, comment style |
| `make typecheck` | `go build` and `tsc` |
| `make test` | Go and Vitest unit tests |
| `make check` | Full pre-merge gate (everything above + infra checks + e2e) |
| `make db-reset` | Wipe, recreate, and migrate the local database |
| `make db-psql` | Open a `psql` shell into the local database |
| `make health` | Curl the API's `/readyz` |

For the test suites in depth, see [testing.md](testing.md).
