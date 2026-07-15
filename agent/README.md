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
Each message may carry an optional `id` (its caller-side row id); compaction echoes
the newest folded one back so the caller can persist a rolling summary.
`overrides` is optional (`model`, `max_searches`, `max_tokens`). An optional `dev`
bool traces the run to `LANGSMITH_PROJECT_DEV` instead of the default project.

## Run events

`POST /run` streams the run as an ordered log of Server-Sent Events. Every step
(reasoning, tool activity, answer text) is a node with a stable `id`:

| Event      | Payload                                | Meaning                                        |
| ---------- | -------------------------------------- | ---------------------------------------------- |
| `node`     | `{id, parent_id, type, name?, input?}` | A step begins: `type` is `reasoning`, `text`, `tool`, or `compaction` |
| `delta`    | `{id, text}`                           | Text appended to a `reasoning` or `text` node  |
| `node_end` | `{id, output?}`                        | A step completes                               |
| `usage`    | `{input, output, total, reasoning}`    | Aggregate token usage across all model calls   |
| `meta`     | `{langsmith_run_id}`                   | The trace's root run id, when tracing is on    |
| `error`    | `{message}`                            | Run failed                                     |
| `end`      | `{}`                                   | Stream complete                                |

A node's `parent_id` is the `id` of the `tool` it runs inside, or `null` at the
top level. A subagent (`run_subagent`) and its own `internet_search` calls
nest under it, so the run reads as a tree.

A `compaction` node marks where older history was summarized to keep the run
within the model's context window. It streams the summary as `delta` text and is
not part of the answer; the web client renders it as its own collapsible step. Its
`node_end` carries `{"summary_through_id"}`, the newest folded message id, which the
caller can store with the summary as a watermark.

The `meta` event's `langsmith_run_id` is the trace root. The API keeps it on the
saved reply so a later thumbs up/down can attach user feedback to that trace.

## Structure

| Path              | Contents                                            |
| ----------------- | --------------------------------------------------- |
| `src/graphs/`     | The `agent` graph and its `subagent`                |
| `src/middleware/` | Cross-cutting graph steps                           |
| `src/tools/`      | Web search, subagent delegation, todo list          |
| `src/prompts/`    | System prompts                                      |
| `src/server.py`   | FastAPI app: `/run`, `/healthz`                     |
| `src/events.py`   | LangGraph events to SSE translation                 |
| `src/config.py`   | Settings and per-run config                         |
| `evals/`          | Live behavioral evals, offline and online (see [evals/README.md](evals/README.md)) |
