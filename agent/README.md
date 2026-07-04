# agent

> 🚧 Work in progress. Standalone service, not yet wired into the API or infra.

A LangGraph agent (web search plus subagents) served over one streaming
endpoint. A generic assistant, not a research-report product.

## Setup

Requires [uv](https://docs.astral.sh/uv/). Run from `agent/`:

```bash
cp .env.example .env    # OpenRouter, Tavily, LangSmith keys
uv sync
uv run uvicorn src.server:app --port 8081   # serve /run on :8081
uv run langgraph dev                         # or open LangGraph Studio
```

## Endpoints

| Method | Path       | Description                                       |
| ------ | ---------- | ------------------------------------------------- |
| `POST` | `/run`     | Run the agent on a message history, stream as SSE |
| `GET`  | `/healthz` | Liveness probe                                    |

Request body: `{"messages": [{"role": "user", "content": "..."}], "overrides": {...}}`.
`overrides` is optional (`model`, `max_searches`, `max_tokens`).

## Run events

`POST /run` streams Server-Sent Events:

| Event       | Payload                              | Meaning                                            |
| ----------- | ------------------------------------ | -------------------------------------------------- |
| `token`     | `{text}`                             | Answer delta                                       |
| `reasoning` | `{text}`                             | Reasoning delta                                    |
| `status`    | `{id, kind, detail, state}`          | Tool activity: `research`, `search`, or `plan`     |
| `usage`     | `{input, output, total, reasoning}`  | Aggregate token usage across all model calls       |
| `error`     | `{message}`                          | Run failed                                         |
| `end`       | `{}`                                 | Stream complete                                    |

`status.state` is `start` or `end`; `id` ties a start to its end, so parallel
tools never cross talk.

## Structure

| Path            | Contents                                     |
| --------------- | -------------------------------------------- |
| `src/graphs/`   | The `agent` graph and its `subagent`         |
| `src/tools/`    | Web search, subagent delegation, todo list   |
| `src/prompts/`  | System prompts                               |
| `src/server.py` | FastAPI app: `/run`, `/healthz`              |
| `src/events.py` | LangGraph events to SSE translation          |
| `src/config.py` | Settings and per-run config                  |

## Tests

```bash
uv run pytest          # unit tests
uv run ruff check .    # lint
```
