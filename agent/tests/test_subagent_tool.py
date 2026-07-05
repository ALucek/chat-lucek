from langchain_core.messages import AIMessage

from src.tools import subagent_tool as st


class _FakeSub:
    def __init__(self):
        self.config = None

    async def ainvoke(self, inp, config=None):
        self.config = config
        return {"messages": [AIMessage(content="summary")]}


async def test_run_subagent_returns_last_message_content(monkeypatch):
    fake = _FakeSub()
    monkeypatch.setattr(st, "subagent", fake)
    result = await st._run_subagent("do research", {})
    assert result == "summary"


async def test_run_subagent_sets_subagent_recursion_limit(monkeypatch):
    from src.config import get_settings

    fake = _FakeSub()
    monkeypatch.setattr(st, "subagent", fake)
    await st._run_subagent("task", {})
    assert fake.config["recursion_limit"] == get_settings().subagent_recursion_limit


def test_build_subagent_tool_naming():
    assert st.build_subagent_tool().name == "run_subagent"
