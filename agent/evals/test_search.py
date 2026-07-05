import json

import pytest
from langchain_core.messages import HumanMessage
from langsmith import testing as t

from evals.harness import judge, load_state, run_step, tool_names
from src.graphs.subagent import subagent

SUITE = "chat-lucek-search"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_returns_cleanly_after_limit_message():
    # Easy: the subagent has already been told the limit is reached.
    state = load_state("search_limit_reached.json")
    t.log_inputs({"case": "limit message seen", "search_count": state["search_count"]})
    reply = await run_step(subagent, state)
    t.log_outputs({"content": reply.content, "tool_calls": sorted(tool_names(reply))})
    assert "internet_search" not in tool_names(reply), (
        "subagent searched again after being told the limit was reached"
    )


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_returns_cleanly_when_budget_spent():
    # Hard: five searches done with real results, no limit message yet
    state = load_state("search_budget_spent.json")
    t.log_inputs(
        {
            "case": "budget spent, no limit message",
            "search_count": state["search_count"],
        }
    )
    reply = await run_step(subagent, state)
    t.log_outputs({"content": reply.content, "tool_calls": sorted(tool_names(reply))})
    assert "internet_search" not in tool_names(reply), (
        "subagent kept searching after spending its budget"
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
                "operators, boolean AND/OR, wildcards) and no filter parameters "
                "beyond the query itself."
            ),
        },
        reference="current population of Tokyo",
        prefix="search_query",
    )
    failed = [r for r in verdict.results if not r.passed]
    assert not failed, [f"{r.assertion}: {r.reasoning}" for r in failed]


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_search_query_is_relevant_to_the_question():
    # Content judge on RELEVANCE: the query is clearly about what was asked.
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
            "relevant": (
                "The search query is clearly related to the user's question and "
                "searches for what they asked about."
            ),
        },
        reference="top tourist attractions in Kyoto",
        prefix="search_query",
    )
    failed = [r for r in verdict.results if not r.passed]
    assert not failed, [f"{r.assertion}: {r.reasoning}" for r in failed]
