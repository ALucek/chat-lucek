# web

The Next.js client for [chat-lucek](../README.md)

## Setup

Requires Node 20+ and pnpm. It expects the [api](../api/README.md) running on `:8080`. Run from the repo root:

```bash
make web-install    # install dependencies
make web-run        # dev server on :3000
```

## Pages

| Path       | Page                |
| ---------- | ------------------- |
| `/login`   | Sign in with Google |
| `/`        | New chat            |
| `/c/{id}`  | A conversation      |
| `/privacy` | Privacy policy      |
| `/terms`   | Terms of service    |

## Structure

| Path              | Contents                                                  |
| ----------------- | --------------------------------------------------------- |
| `src/app/`        | App Router pages, grouped by `(auth)`, `(app)`, `(legal)` |
| `src/components/` | UI components                                             |
| `src/lib/`        | API client, context providers, and hooks                  |
