## Setup

Requires Go 1.26+ and Docker (for local Postgres). Run from the repo root:

```bash
cp .env.example .env    # fill in Google and JWT secrets
make db-up              # start Postgres
make migrate-up         # apply migrations
make api-run            # serve on :8080
```

Chat runs through the [agent](../agent/) service; start it too, and point `AGENT_URL` at it (default `http://localhost:8081`).

## Routes

Everything under `/api/` requires an `Authorization: Bearer <access token>` header, except sign-in, refresh, and logout.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/livez` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (checks the database) |
| `GET` | `/agentz` | api-to-agent integration check (dev host only) |
| `POST` | `/api/google` | Sign in with a Google auth code, returns tokens |
| `POST` | `/api/magic/request` | Email a single-use magic sign-in link |
| `POST` | `/api/magic/verify` | Verify a magic-link token, returns tokens |
| `POST` | `/api/refresh` | Exchange a refresh token for a new access token |
| `POST` | `/api/logout` | Revoke the refresh token |
| `GET` | `/api/me` | Current user |
| `GET` | `/api/conversations` | List the current user's conversations |
| `POST` | `/api/conversations` | Create a conversation |
| `GET` | `/api/conversations/{id}/messages` | Message history |
| `PATCH` | `/api/conversations/{id}` | Rename a conversation |
| `DELETE` | `/api/conversations/{id}` | Delete a conversation |
| `POST` | `/api/conversations/{id}/messages` | Send a message, stream the reply (SSE) |
| `GET` | `/api/usage` | Current user's run count against the daily budget |

## Schema

| Table | Columns |
| --- | --- |
| `users` | `id`, `email`, `created_at` |
| `refresh_tokens` | `token_hash`, `user_id`, `family_id`, `expires_at`, `revoked`, `created_at` |
| `magic_links` | `token_hash`, `email`, `expires_at`, `created_at` |
| `conversations` | `id`, `user_id`, `title`, `created_at`, `updated_at` |
| `messages` | `id`, `conversation_id`, `role`, `content`, `trace`, `created_at` |
| `token_usage` | `id`, `user_id`, `prompt_tokens`, `completion_tokens`, `created_at` |
| `usage_marks` | `id`, `subject_hash`, `created_at` |

The `user_id` and `conversation_id` columns are foreign keys to their parent tables. Deletes cascade down the `users` > `conversations` > `messages` chain.