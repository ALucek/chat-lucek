# Testing

Three test layers, all runnable with `make`. CI runs the same targets you run locally, so a green `make check` predicts a green pipeline.

## Suites

| Suite | Command | What it covers |
| --- | --- | --- |
| Go (api) | `make api-test` | Handler and DB logic against a throwaway Postgres (testcontainers, needs Docker) |
| Web | `make web-test` | Vitest unit and component tests |
| End-to-end | `make e2e-local` | Playwright drives the full stack in a real browser |

The Go suite starts one Postgres container, migrates it, and truncates tables between tests, so no local database is required. The e2e suite uses **fake** Google Sign-In and OpenRouter servers, so it needs no real credentials or network.

First-time e2e run needs the browser installed once:

```bash
cd web && pnpm exec playwright install chromium
```

`make e2e-local` starts Postgres, migrates, and runs the suite; Playwright boots the api, web, and fake servers itself. Use `make e2e` on its own if the database is already up and migrated.

## Gates

| Command | Runs |
| --- | --- |
| `make test` | The Go and web unit suites |
| `make check` | The full pre-merge gate: format, lint, typecheck, all suites, infra and workflow checks, and e2e |

## CI

Every push and pull request runs [test.yml](../.github/workflows/test.yml). Its `api-test`, `web-test`, and `e2e` jobs run the same make targets as above, and passing them is required to merge.
