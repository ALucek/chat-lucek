## Setup

Requires [uv](https://docs.astral.sh/uv/). Run from `agent/`:

```bash
cp .env.example .env    # OpenRouter, Tavily, LangSmith keys
uv sync
uv run uvicorn src.server:app --port 8081   # serve /run on :8081
uv run langgraph dev                         # or open LangGraph Studio
```

The API reaches the agent at `AGENT_URL` (default `http://localhost:8081`).

## Endpoints

| Method | Path       | Description                                       |
| ------ | ---------- | ------------------------------------------------- |
| `POST` | `/run`     | Run the agent on a message history, stream as SSE |
| `GET`  | `/healthz` | Liveness probe                                    |

Request body: `{"messages": [{"role": "user", "content": "..."}], "overrides": {...}}`.
`overrides` is optional (`model`, `max_searches`, `max_tokens`).

## Run events

`POST /run` streams the run as an ordered log of Server-Sent Events. Every step
(reasoning, tool activity, answer text) is a node with a stable `id`:

| Event      | Payload                                | Meaning                                        |
| ---------- | -------------------------------------- | ---------------------------------------------- |
| `node`     | `{id, parent_id, type, name?, input?}` | A step begins: `type` is `reasoning`, `text`, or `tool` |
| `delta`    | `{id, text}`                           | Text appended to a `reasoning` or `text` node  |
| `node_end` | `{id, output?}`                        | A step completes                               |
| `usage`    | `{input, output, total, reasoning}`    | Aggregate token usage across all model calls   |
| `error`    | `{message}`                            | Run failed                                     |
| `end`      | `{}`                                   | Stream complete                                |

A node's `parent_id` is the `id` of the `tool` it runs inside, or `null` at the
top level. A subagent (`run_subagent`) and its own `internet_search` calls
nest under it, so the run reads as a tree.

## Structure

| Path            | Contents                                     |
| --------------- | -------------------------------------------- |
| `src/graphs/`   | The `agent` graph and its `subagent`         |
| `src/tools/`    | Web search, subagent delegation, todo list   |
| `src/prompts/`  | System prompts                               |
| `src/server.py` | FastAPI app: `/run`, `/healthz`              |
| `src/events.py` | LangGraph events to SSE translation          |
| `src/config.py` | Settings and per-run config                  |
