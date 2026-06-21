# api — simple-ai-chatbot backend

A small Go HTTP API for a multi-user streaming chatbot: hand-rolled JWT auth,
per-user conversations, and replies streamed from an LLM over Server-Sent Events.

## Stack

- **Go** `net/http` (1.22 method+path routing), `pgx`/`pgxpool`
- **Postgres** (local via Docker), schema managed with **goose** migrations
- **Auth**: `bcrypt` + HS256 JWT access tokens + DB-backed refresh tokens
- **LLM**: OpenRouter (OpenAI-compatible) streamed as raw SSE
- **Tests**: `testcontainers-go` against a real ephemeral Postgres

## Quick start

From the repo root (the `Makefile` and `.env` live there):

```bash
cp .env.example .env        # then fill in OPENROUTER_API_KEY and a JWT_SECRET
make db-up                  # start Postgres in Docker
make migrate-up             # apply migrations
make run                    # start the API on :$PORT (default 8080)
make health                 # -> 200 when the DB is reachable
```

Generate a secret with `openssl rand -hex 32`. Get a key (and pick a model) at
[openrouter.ai](https://openrouter.ai); the default model is a free slug.

## Configuration

Read from the environment (see `.env.example`):

| Var | Required | Default | Notes |
|---|---|---|---|
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` | yes | — | Postgres connection |
| `PORT` | yes | — | HTTP listen port |
| `JWT_SECRET` | yes | — | HS256 signing key |
| `OPENROUTER_API_KEY` | yes | — | LLM access |
| `OPENROUTER_MODEL` | no | `openrouter/free` | any openrouter.ai/models slug |
| `SYSTEM_PROMPT` | no | `You are a helpful assistant.` | prepended to every request |

## API

All `/api/*` routes except signup/login/refresh require
`Authorization: Bearer <access_token>`.

| Method & path | Purpose |
|---|---|
| `GET /health` | DB-backed liveness check |
| `POST /api/signup` | create user → `{access_token, refresh_token}` |
| `POST /api/login` | verify password → `{access_token, refresh_token}` |
| `POST /api/refresh` | exchange refresh token for a new access token |
| `POST /api/logout` | revoke a refresh token |
| `GET /api/me` | current user |
| `GET /api/conversations` | list the caller's conversations |
| `POST /api/conversations` | create a conversation |
| `GET /api/conversations/{id}/messages` | message history |
| `PATCH /api/conversations/{id}` | rename |
| `DELETE /api/conversations/{id}` | delete (messages cascade) |
| `POST /api/conversations/{id}/messages` | send a message, **stream** the reply |

### Streaming

`POST /api/conversations/{id}/messages` responds with `text/event-stream`. Each
frame's `data:` is a JSON object; read the stream to EOF:

```
event: delta
data: {"text":"Hel"}

event: done
data: {"message_id":42}

event: title
data: {"title":"Plan a trip to"}
```

- `delta` — an incremental chunk of the reply
- `done` — the reply is complete and persisted (`message_id`)
- `title` — on a conversation's first message only, its generated name (may follow `done`)
- `error` — something failed mid-stream (`{"error":"..."}`)

## Tests

```bash
make test     # or: cd api && go test ./...
```

Integration tests run the real handlers against a throwaway Postgres
(Docker required) and cover the auth flow, CRUD, per-user scoping (IDOR), and
streaming. CI runs the same suite on every push.

## Layout

```
api/
  main.go          # wiring, routes (newMux), graceful shutdown
  config.go        # env config
  db.go            # pgxpool connection + health check
  auth.go          # bcrypt, JWT, refresh tokens, middleware
  auth_handlers.go # signup / login / refresh / logout / me
  chat.go          # conversations CRUD + streaming + titles
  openrouter.go    # OpenRouter SSE client
  migrations/      # goose SQL migrations
  *_test.go        # unit + integration tests
```
