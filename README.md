# simple-ai-chatbot

A small, full-stack, multi-user streaming chatbot built from the ground up. The Go API calls the model's HTTP API directly and hand-rolls the streaming: it reads the raw upstream Server-Sent Events and re-streams them to a hand-built Next.js client that renders the reply token-by-token. Everything in between (auth, conversation storage, the SSE plumbing) is plain Go and plain React.

WIP learning project to get the ropes with fullstack development!

## Components

| Dir | What it is | README |
|---|---|---|
| [`api/`](api/README.md) | Go HTTP API — JWT auth, per-user conversations, LLM streaming over SSE | [api/README.md](api/README.md) |
| [`web/`](web/README.md) | Next.js client — auth, conversation sidebar, streamed chat | [web/README.md](web/README.md) |

## Stack

- **Backend**: Go `net/http`, `pgx`/`pgxpool`, `bcrypt` + JWT, OpenRouter (OpenAI-compatible)
- **Database**: Postgres (local via Docker), `goose` migrations
- **Frontend**: Next.js 16 (App Router, TS strict), React 19, Tailwind v4, pnpm
- **Tests**: `testcontainers-go` (API, real Postgres) and Vitest + RTL (web); Playwright e2e; CI on every push
- **Deploy**: multi-stage Docker images (distroless API, Next standalone web); full local stack via `docker compose` profiles

## Quick start

Prerequisites: Docker, Go, and pnpm.

```bash
cp .env.example .env        # then fill in OPENROUTER_API_KEY and a JWT_SECRET
make db-up                  # start Postgres in Docker
make migrate-up             # apply migrations
make api-run                # start the API on :$PORT (default 8080)

# in a second terminal:
make web-install            # install web dependencies
make web-run                # start the client on :3000
```

Then open <http://localhost:3000> and sign up. Generate a JWT secret with
`openssl rand -hex 32`; get an API key (and pick a model) at [openrouter.ai](https://openrouter.ai).

## Production-parity stack

The Quick start runs the services with hot reload. To instead run the **built** container
images — a small, non-root pair (distroless Go binary for the API, Next `standalone` server
for the web) — as a full local stack:

```bash
make docker-build           # build both images
make stack-up               # start db + api + web (compose `full` profile), then migrate
# open http://localhost:3000
make stack-down             # tear the stack down
```

`make db-up` is unchanged — it still starts **only** Postgres for hot-reload dev; the `api`
and `web` services sit behind a compose `full` profile. Both images take all configuration
from environment variables, so the same images run in the cloud, differing only by what's
injected. `NEXT_PUBLIC_API_URL` is the exception: Next bakes public vars at build time, so
it's a Docker build-arg.

Configuration lives in the repo-root `.env` (consumed by both the `Makefile` and the API);
see [`api/README.md`](api/README.md#configuration) for the full variable list and
[`web/README.md`](web/README.md#configuration) for the single client variable.
