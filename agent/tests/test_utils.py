from langchain_core.messages import AIMessage, HumanMessage, SystemMessage

from src.utils import build_messages


def test_prepends_system_prompt():
    out = build_messages("SYS", [HumanMessage(content="hi")])
    assert isinstance(out[0], SystemMessage)
    assert out[0].content == "SYS"
    assert out[1:] == [HumanMessage(content="hi")]


def test_preserves_full_working_state_without_trimming():
    # No windowing here: the agent must keep its whole scratchpad intact.
    history: list = []
    for i in range(30):
        history.append(HumanMessage(content=f"h{i}"))
        history.append(AIMessage(content=f"a{i}"))
    out = build_messages("SYS", history)
    assert isinstance(out[0], SystemMessage)
    assert out[1:] == history  # nothing dropped
    assert out[1].content == "h0"  # original first turn retained
