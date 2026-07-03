# api

The Go backend for [chat-lucek](../README.md)

## Setup

Requires Go 1.26+ and Docker (for local Postgres). Run from the repo root:

```bash
cp .env.example .env    # fill in OpenRouter, Google, and JWT secrets
make db-up              # start Postgres
make migrate-up         # apply migrations
make api-run            # serve on :8080
```

## Routes

Everything under `/api/` requires an `Authorization: Bearer <access token>` header, except sign-in, refresh, and logout.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/livez` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (checks the database) |
| `POST` | `/api/google` | Sign in with a Google auth code, returns tokens |
| `POST` | `/api/refresh` | Exchange a refresh token for a new access token |
| `POST` | `/api/logout` | Revoke the refresh token |
| `GET` | `/api/me` | Current user |
| `GET` | `/api/conversations` | List the current user's conversations |
| `POST` | `/api/conversations` | Create a conversation |
| `GET` | `/api/conversations/{id}/messages` | Message history |
| `PATCH` | `/api/conversations/{id}` | Rename a conversation |
| `DELETE` | `/api/conversations/{id}` | Delete a conversation |
| `POST` | `/api/conversations/{id}/messages` | Send a message, stream the reply (SSE) |
| `GET` | `/api/usage` | Current user's token usage and budget |

## Schema

| Table | Columns |
| --- | --- |
| `users` | `id`, `google_sub`, `email`, `created_at` |
| `refresh_tokens` | `token_hash`, `user_id`, `family_id`, `expires_at`, `revoked`, `created_at` |
| `conversations` | `id`, `user_id`, `title`, `created_at`, `updated_at` |
| `messages` | `id`, `conversation_id`, `role`, `content`, `created_at` |
| `token_usage` | `id`, `user_id`, `prompt_tokens`, `completion_tokens`, `created_at` |

The `user_id` and `conversation_id` columns are foreign keys to their parent tables. Deletes cascade down the `users` > `conversations` > `messages` chain.