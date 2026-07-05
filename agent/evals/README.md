# Agent evals

Behavioral evals for the agent, run on demand against the real models and synced to LangSmith. They call the live agent (OpenRouter) and Tavily, so unlike `agent/tests/` they cost money and stay out of the CI gate. Philosophy: https://lucek.ai/blogs/agent-evaluations

## Layout

Flat, one file per feature. Each file is one LangSmith dataset, set by `@pytest.mark.langsmith(test_suite_name=...)`.

| File | Dataset | Covers |
| --- | --- | --- |
| `test_routing.py` | `chat-lucek-routing` | planning and subagent delegation |
| `test_search.py` | `chat-lucek-search` | search-limit handling and query quality |
| `test_tone.py` | `chat-lucek-tone` | answer tone |

Each test is one dataset example, keyed by its logged inputs, so distinct behaviors need distinct inputs.

## harness.py

The shared toolkit:

- `load_state(name)` loads a JSON trace fixture from `fixtures/` into a graph state. Fixtures are real captured runs; each records its source run id.
- `run_step(graph, state)` drives a compiled graph one node and returns the message it produced, so step evals see the agent's next decision without running a tool.
- `tool_names(message)` returns the tool-call names on a message.
- `judge(content, asserts, *, prefix, reference=None)` is the LLM judge. `asserts` is a `{name: assertion}` dict; it grades each with reasoning then a boolean and logs one score per assertion as `<prefix>_<name>`. `reference`, when given, is shown as an example of an ideal response. The judge model is `JUDGE_MODEL` (`anthropic/claude-sonnet-5`), overridable with `EVAL_JUDGE_MODEL`.

## Running

Keys load from `agent/.env`. From the repo root:

| Command | Use |
| --- | --- |
| `make evals` | Authoritative: uncached, real models, syncs to LangSmith |
| `make evals-cached` | Iteration: replays cassettes in `evals/cassettes` |

Cassettes key on the request, so changing a prompt, model, tool schema, or input re-records. Cached runs replay frozen responses and miss model drift, so trust `make evals` for real numbers. Cassettes are gitignored.

## Adding an eval

Add to the feature file it belongs to, or start a new `test_<feature>.py` (one file per dataset) marked with `test_suite_name`. Log `t.log_inputs`/`t.log_outputs`, and `t.log_reference_outputs` when there is a gold example. Step evals assert on `tool_names(await run_step(graph, state))`; content evals run the full graph and gate on `judge(...)`. Seed fixtures from real LangSmith traces, and keep assertions grounded in what the prompt actually specifies.
