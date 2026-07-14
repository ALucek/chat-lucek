import pytest
from langchain_core.messages import AIMessage, HumanMessage

from src.graphs import subagent as sub_mod
from src.graphs.subagent import route_subagent


def _search_call(query, call_id):
    return {"name": "internet_search", "args": {"query": query}, "id": call_id}


def test_route_to_tools_when_tool_calls_present():
    msg = AIMessage(content="", tool_calls=[_search_call("q", "1")])
    assert route_subagent({"messages": [msg]}) == "tools"


def test_route_ends_without_tool_calls():
    assert route_subagent({"messages": [AIMessage(content="answer")]}) == "end"


class _FakeSearch:
    def __init__(self, responses):
        self._responses = list(responses)
        self.calls = []

    async def ainvoke(self, args):
        self.calls.append(args)
        result = self._responses.pop(0)
        if isinstance(result, Exception):
            raise result
        return result


@pytest.fixture
def fake_search(monkeypatch):
    def _install(responses):
        fake = _FakeSearch(responses)
        monkeypatch.setattr(sub_mod, "web_search_tool", fake)
        return fake

    return _install


async def test_web_search_node_runs_and_counts(fake_search):
    fake_search(["## Search results for: q\n\n1. [T](u)\n\t- c\n"])
    msg = AIMessage(content="", tool_calls=[_search_call("q", "1")])
    out = await sub_mod.web_search_node({"messages": [msg], "search_count": 0}, {})
    assert out["search_count"] == 1
    assert "[T](u)" in out["messages"][0].content


async def test_web_search_node_caps_at_max_searches(fake_search):
    fake_search([])  # nothing executes; all calls are over the limit
    calls = [_search_call(f"q{i}", str(i)) for i in range(3)]
    msg = AIMessage(content="", tool_calls=calls)
    out = await sub_mod.web_search_node({"messages": [msg], "search_count": 5}, {})
    assert out["search_count"] == 5  # unchanged; default max_searches is 5
    assert all("Search limit reached" in m.content for m in out["messages"])


async def test_web_search_node_reports_errors(fake_search):
    fake_search([RuntimeError("boom")])
    msg = AIMessage(content="", tool_calls=[_search_call("q", "1")])
    out = await sub_mod.web_search_node({"messages": [msg], "search_count": 0}, {})
    assert "Error running search: boom" in out["messages"][0].content


async def test_web_search_node_passes_tool_result_through(fake_search):
    fake_search(["## Search results for: q\n\n1. [T](u)"])
    msg = AIMessage(content="", tool_calls=[_search_call("q", "1")])
    out = await sub_mod.web_search_node({"messages": [msg], "search_count": 0}, {})
    assert out["messages"][0].content == "## Search results for: q\n\n1. [T](u)"


class _FakeModel:
    def bind_tools(self, tools):
        return self

    async def ainvoke(self, messages, config=None):
        return AIMessage(content="findings")


async def test_subagent_node_wraps_model_response(monkeypatch):
    monkeypatch.setattr(
        sub_mod, "build_chat_model", lambda cfg, role=None, **kw: _FakeModel()
    )
    out = await sub_mod.subagent_node(
        {"messages": [HumanMessage(content="hi")], "search_count": 0}, {}
    )
    assert [m.content for m in out["messages"]] == ["findings"]
