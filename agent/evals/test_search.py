import json

import pytest
from langchain_core.messages import HumanMessage
from langsmith import testing as t

from evals.harness import judge, load_state, run_step, tool_names
from src.graphs.subagent import subagent

SUITE = "chat-lucek-search"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_returns_cleanly_when_search_limit_reached():
    # Pre-set trace: budget already spent, last result reports the limit.
    state = load_state("search_limit_reached.json")
    t.log_inputs(
        {"search_count": state["search_count"], "turns": len(state["messages"])}
    )
    reply = await run_step(subagent, state)
    t.log_outputs({"content": reply.content, "tool_calls": sorted(tool_names(reply))})
    assert "internet_search" not in tool_names(reply), (
        "subagent retried search after the limit"
    )


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_search_uses_only_a_query_arg():
    # Structural guard: the tool call carries a query and no filter params.
    question = "What is the current population of Osaka?"
    state = {"messages": [HumanMessage(content=question)], "search_count": 0}
    t.log_inputs({"question": question})
    reply = await run_step(subagent, state)
    calls = [c for c in reply.tool_calls if c["name"] == "internet_search"]
    t.log_outputs({"tool_call_args": [c["args"] for c in calls]})
    assert calls, "expected the subagent to run a web search"
    for call in calls:
        assert set(call["args"]) <= {"query"}, (
            f"search call carried filter args: {call['args']}"
        )
        assert (
            isinstance(call["args"].get("query"), str) and call["args"]["query"].strip()
        )


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_search_query_is_clean_natural_language():
    # Content judge on query FORM: a bare, operator-free natural-language query.
    question = "What is the current population of Tokyo?"
    state = {"messages": [HumanMessage(content=question)], "search_count": 0}
    t.log_inputs({"question": question})
    t.log_reference_outputs({"ideal_query": "current population of Tokyo"})

    reply = await run_step(subagent, state)
    calls = [c for c in reply.tool_calls if c["name"] == "internet_search"]
    assert calls, "expected the subagent to run a web search"
    args = calls[0]["args"]
    t.log_outputs({"tool_call_args": args})

    content = f"User question: {question}\nSearch tool call args: {json.dumps(args)}"
    verdict = judge(
        content,
        {
            "natural_language": (
                "The search input is a single natural-language query with no "
                "search-engine operators or special syntax (no site:, quoted "
                "operators, boolean AND/OR, wildcards)."
            ),
            "no_filters": (
                "No filter parameters are set beyond the query itself (no time "
                "range, domain include/exclude, topic override, result count)."
            ),
        },
        reference="current population of Tokyo",
        prefix="search_query",
    )
    failed = [r for r in verdict.results if not r.passed]
    assert not failed, [f"{r.assertion}: {r.reasoning}" for r in failed]


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_search_query_is_faithful_to_the_question():
    # Content judge on FAITHFULNESS: the query adds nothing the user did not ask.
    question = "What are the top tourist attractions in Kyoto?"
    state = {"messages": [HumanMessage(content=question)], "search_count": 0}
    t.log_inputs({"question": question})
    t.log_reference_outputs({"ideal_query": "top tourist attractions in Kyoto"})

    reply = await run_step(subagent, state)
    calls = [c for c in reply.tool_calls if c["name"] == "internet_search"]
    assert calls, "expected the subagent to run a web search"
    args = calls[0]["args"]
    t.log_outputs({"tool_call_args": args})

    content = f"User question: {question}\nSearch tool call args: {json.dumps(args)}"
    verdict = judge(
        content,
        {
            "faithful": (
                "The query faithfully reflects the user's question without adding "
                "topics, entities, dates, or constraints the user did not ask for."
            ),
        },
        reference="top tourist attractions in Kyoto",
        prefix="search_query",
    )
    failed = [r for r in verdict.results if not r.passed]
    assert not failed, [f"{r.assertion}: {r.reasoning}" for r in failed]
