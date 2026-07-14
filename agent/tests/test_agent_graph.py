from langchain_core.messages import AIMessage, HumanMessage

from src.graphs import agent as agent_mod
from src.graphs.agent import route_agent


def _ai_with_tools(names):
    return AIMessage(
        content="",
        tool_calls=[{"name": n, "args": {}, "id": str(i)} for i, n in enumerate(names)],
    )


def test_route_ends_without_tool_calls():
    assert route_agent({"messages": [AIMessage(content="done")]}) == "end"


def test_route_ends_when_last_message_not_ai():
    assert route_agent({"messages": [HumanMessage(content="hi")]}) == "end"


def test_route_to_tools_when_tool_calls_present():
    assert route_agent({"messages": [_ai_with_tools(["run_subagent"])]}) == "tools"


def test_graph_binds_both_tools():
    assert {t.name for t in agent_mod.tools} == {"run_subagent", "set_todos"}


class _FakeModel:
    def bind_tools(self, tools):
        return self

    async def ainvoke(self, messages, config=None):
        return AIMessage(content="answer")


async def test_agent_node_wraps_model_response(monkeypatch):
    # Wiring only: node builds a model, binds tools, wraps the response.
    monkeypatch.setattr(
        agent_mod, "build_chat_model", lambda cfg, role=None, **kw: _FakeModel()
    )
    out = await agent_mod.agent_node({"messages": [HumanMessage(content="hi")]}, {})
    assert [m.content for m in out["messages"]] == ["answer"]
