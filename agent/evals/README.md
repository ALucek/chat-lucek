# Agent evals

Live behavioral evals for the agent, built on the LangSmith pytest integration. They call the real agent and sync to LangSmith, so unlike `agent/tests/` they cost money and stay out of the CI gate.

## Running

Keys load from `agent/.env`. From the repo root:

| Command | Runs |
| --- | --- |
| `make evals` | The eval suite uncached against real models |
| `make evals-cached` | The eval suite replaying cassettes in `evals/cassettes` |

Scope either target to one file, suite, or test with `E`, a pytest selector under `evals/`:

```bash
make evals E=test_ability.py                          # one file (= one suite)
make evals E=test_ability.py::test_finds_the_weather  # one eval
make evals E="test_ability.py -k weather"             # by keyword within a file
```

Cassettes key on the request, so changing a prompt, model, tool schema, or input re-records. Cached runs replay frozen responses and miss model drift, so trust `make evals` for real numbers.

The [`evals` workflow](../../.github/workflows/evals.yml) also runs the full suite every Monday (and on demand from the Actions tab) and emails the output to the owner via Resend.

## Evals

**[Routing](test_routing.py)** (`chat-lucek-routing`) — does the main agent pick the right next step? Step evals on the router.

| Test | Checks |
| --- | --- |
| `plans_multi_step_task` | calls `set_todos` on a clearly multi-step task |
| `no_plan_for_trivial_request` | no `set_todos` on a trivial one-step question |
| `delegates_research_to_subagent` | calls `run_subagent` for current-info research |
| `answers_simple_question_directly` | no `run_subagent` for a directly answerable question |

**[Search](test_search.py)** (`chat-lucek-search`) — how the subagent handles its search budget and shapes queries.

| Test | Checks |
| --- | --- |
| `returns_cleanly_after_limit_message` | stops searching once told the limit is reached |
| `returns_cleanly_when_budget_spent` | stops on its own after the budget is spent, no limit message |
| `search_uses_only_a_query_arg` | the search call carries only a `query`, no filter params |
| `search_query_is_clean_natural_language` | a bare natural-language query, no operators or filters |
| `search_query_is_relevant_to_the_question` | the query is clearly about what was asked |

**[Tone](test_tone.py)** (`chat-lucek-tone`) — does the answer follow the prompt's `<tone>`/`<behavior>`? One judged criterion each.

| Test | Checks |
| --- | --- |
| `tone_is_plain_and_direct` | plain and direct, no padding |
| `tone_has_no_filler` | no filler pleasantries, forced enthusiasm, or greeting fluff |
| `tone_does_not_perform` | no sycophancy, overeagerness, emoji spam, or AI-isms |

**[Ability](test_ability.py)** (`chat-lucek-ability`) — end-to-end full-agent runs.

| Test | Checks |
| --- | --- |
| `test_knows_its_identity` | knows it is Harold (code assert) |
| `test_completes_a_web_search` | delegates to the subagent, searches, and returns a live answer (LLM-judge) |

## Online evals

Separate from the suite above: online evaluators that score live prod traces server-side in LangSmith. Definitions live in `online/`, provisioned as IaC in [`infra/langsmith.tf`](../../infra/langsmith.tf).

| Evaluator | Type | Scores | Feedback |
| --- | --- | --- | --- |
| [`overcapped_searches.js`](online/overcapped_searches.js) | code | subagent runs | `overcapped_searches` (search calls attempted past the cap) |
| [`pii_scan.js`](online/pii_scan.js) | code | final answers | `pii_detected` (binary) |
| [`conversation_length.js`](online/conversation_length.js) | code | root runs | `conversation_length` (user turns) |
| [`prompt_injection.py`](online/prompt_injection.py) | LLM judge | user inputs | `prompt_injection_score` (binary), `prompt_injection_explanation` |
| [`thread_helpfulness.py`](online/thread_helpfulness.py) | LLM judge | whole thread | `thread_helpful` (binary), `thread_helpful_explanation` |

The `.js` evaluators are code that runs in LangSmith's sandbox. LLM-as-judge evaluators are each a self-contained Python file whose prompt, output schema, and model publish to the LangSmith Prompt Hub with `make push-llm-judge JUDGE=<name>`, then get referenced from Terraform by a pinned commit hash.

Some of these scores also drive email alerts to the owner; see [monitoring.md](../../docs/monitoring.md#tracing).