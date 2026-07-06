from langchain_core.messages import AIMessage, HumanMessage, ToolMessage

from evals.harness import load_state, tool_names


def test_load_state_rebuilds_trace_and_search_count():
    state = load_state("search_limit_reached.json")
    assert state["search_count"] == 5
    msgs = state["messages"]
    assert isinstance(msgs[0], HumanMessage)
    # Tool-call messages deserialize with their ids preserved (needed for the
    # ai/tool pairing the model API requires).
    search_calls = [
        c
        for m in msgs
        if isinstance(m, AIMessage)
        for c in m.tool_calls
        if c["name"] == "internet_search"
    ]
    assert search_calls and all(c["id"] for c in search_calls)
    # The trace ends at the search limit.
    assert isinstance(msgs[-1], ToolMessage)
    assert "Search limit reached" in msgs[-1].content


def test_tool_names_extracts_call_names():
    msg = AIMessage(
        content="",
        tool_calls=[
            {"name": "set_todos", "args": {}, "id": "1"},
            {"name": "run_subagent", "args": {}, "id": "2"},
        ],
    )
    assert tool_names(msg) == {"set_todos", "run_subagent"}


def test_tool_names_empty_without_tool_calls():
    assert tool_names(AIMessage(content="just a reply")) == set()
