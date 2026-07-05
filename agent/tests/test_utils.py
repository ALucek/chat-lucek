from langchain_core.messages import AIMessage, HumanMessage, SystemMessage

from src.config import AgentConfig
from src.utils import build_chat_model, build_messages


def test_build_chat_model_applies_overrides(monkeypatch):
    monkeypatch.setenv("OPENROUTER_API_KEY", "test")
    model = build_chat_model(AgentConfig(), role="agent", max_tokens=999)
    assert model.max_tokens == 999


def test_prepends_system_prompt():
    out = build_messages("SYS", [HumanMessage(content="hi")])
    assert isinstance(out[0], SystemMessage)
    assert out[0].content == "SYS"
    assert out[1:] == [HumanMessage(content="hi")]


def test_preserves_full_working_state_without_trimming():
    # The agent must keep its whole scratchpad: trimming mid-run drops the
    # original task and breaks tool-call/result pairs. Windowing is the
    # gateway's job, not the agent's.
    history: list = []
    for i in range(30):
        history.append(HumanMessage(content=f"h{i}"))
        history.append(AIMessage(content=f"a{i}"))
    out = build_messages("SYS", history)
    assert isinstance(out[0], SystemMessage)
    assert out[1:] == history  # nothing dropped
    assert out[1].content == "h0"  # original first turn retained
