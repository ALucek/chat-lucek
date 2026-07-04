from langchain_core.messages import AIMessage, HumanMessage, SystemMessage

from src.config import AgentConfig
from src.utils import MAX_HISTORY_MESSAGES, build_chat_model, build_messages


def test_build_chat_model_applies_overrides(monkeypatch):
    monkeypatch.setenv("OPENROUTER_API_KEY", "test")
    model = build_chat_model(AgentConfig(), role="agent", max_tokens=999)
    assert model.max_tokens == 999


def test_prepends_system_prompt():
    out = build_messages("SYS", [HumanMessage(content="hi")])
    assert isinstance(out[0], SystemMessage)
    assert out[0].content == "SYS"
    assert out[-1].content == "hi"


def test_short_history_kept_intact():
    history = [
        HumanMessage(content="a"),
        AIMessage(content="b"),
        HumanMessage(content="c"),
    ]
    out = build_messages("SYS", history)
    assert [m.content for m in out] == ["SYS", "a", "b", "c"]


def test_long_history_trimmed_to_last_n_and_starts_on_human():
    history = []
    for i in range(20):
        history.append(HumanMessage(content=f"h{i}"))
        history.append(AIMessage(content=f"a{i}"))
    out = build_messages("SYS", history)
    assert isinstance(out[0], SystemMessage)
    assert len(out) <= MAX_HISTORY_MESSAGES + 1  # system + trimmed window
    assert isinstance(out[1], HumanMessage)  # window starts on a human turn
    assert out[-1].content == "a19"  # most recent kept
