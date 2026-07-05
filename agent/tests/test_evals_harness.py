from langchain_core.messages import AIMessage, HumanMessage, ToolMessage

from evals.harness import load_state


def test_load_state_rebuilds_trace_and_search_count():
    state = load_state("search_limit_reached.json")
    assert state["search_count"] == 5
    msgs = state["messages"]
    assert isinstance(msgs[0], HumanMessage)
    ai_with_calls = [m for m in msgs if isinstance(m, AIMessage) and m.tool_calls]
    assert len(ai_with_calls) == 5
    assert ai_with_calls[0].tool_calls[0]["name"] == "internet_search"
    assert isinstance(msgs[-1], ToolMessage)
    assert "Search limit reached" in msgs[-1].content
