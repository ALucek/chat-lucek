from langchain_core.messages import (
    AIMessage,
    HumanMessage,
    RemoveMessage,
    SystemMessage,
    ToolMessage,
)
from langchain_core.messages.utils import count_tokens_approximately
from langgraph.graph.message import REMOVE_ALL_MESSAGES

from src.middleware import summarization as summ
from src.middleware.summarization import (
    SUMMARY_PREFIX,
    _newest_durable_id,
    _split,
    compact,
)


class _FakeSummarizer:
    def __init__(self):
        self.seen = None

    async def ainvoke(self, messages, config=None):
        self.seen = messages
        return AIMessage(content="THE SUMMARY")


class _Boom:
    async def ainvoke(self, messages, config=None):
        raise RuntimeError("summarizer down")


class _CapturingSummarizer:
    def __init__(self):
        self.config = None

    async def ainvoke(self, messages, config=None):
        self.config = config
        return AIMessage(content="THE SUMMARY")


# threshold gates compaction; ratio * threshold is the kept-tail token budget.
def _cfg(threshold, ratio):
    return {
        "configurable": {
            "summary_threshold": threshold,
            "summary_keep_ratio": ratio,
        }
    }


async def test_under_threshold_is_noop():
    state = {"messages": [HumanMessage(content="hi")]}
    assert await compact(state, _cfg(10_000, 0.1)) == {}


async def test_over_threshold_folds_head_keeps_newest(monkeypatch):
    monkeypatch.setattr(
        summ, "build_chat_model", lambda cfg, role=None, **kw: _FakeSummarizer()
    )
    msgs = [
        HumanMessage(content="a"),
        AIMessage(content="b"),
        HumanMessage(content="c"),
        AIMessage(content="d"),
    ]
    # ratio 0.0 -> budget 0 -> tail is just the newest message.
    result = (await compact({"messages": msgs}, _cfg(1, 0.0)))["messages"]
    assert isinstance(result[0], RemoveMessage) and result[0].id == REMOVE_ALL_MESSAGES
    assert isinstance(result[1], HumanMessage) and result[1].content.startswith(
        SUMMARY_PREFIX
    )
    assert [m.content for m in result[2:]] == ["d"]


async def test_summary_receives_prior_summary_for_folding(monkeypatch):
    fake = _FakeSummarizer()
    monkeypatch.setattr(summ, "build_chat_model", lambda cfg, role=None, **kw: fake)
    prior = HumanMessage(content=SUMMARY_PREFIX + "earlier")
    msgs = [
        prior,
        AIMessage(content="b"),
        HumanMessage(content="c"),
        AIMessage(content="d"),
    ]
    await compact({"messages": msgs}, _cfg(1, 0.0))
    assert any(getattr(m, "content", "").startswith(SUMMARY_PREFIX) for m in fake.seen)


async def test_summarizer_gets_instruction_as_last_human_turn(monkeypatch):
    fake = _FakeSummarizer()
    monkeypatch.setattr(summ, "build_chat_model", lambda cfg, role=None, **kw: fake)
    msgs = [
        HumanMessage(content="a"),
        AIMessage(content="b"),
        HumanMessage(content="c"),
        AIMessage(content="d"),
    ]
    await compact({"messages": msgs}, _cfg(1, 0.0))
    assert isinstance(fake.seen[-1], HumanMessage)
    assert not any(isinstance(m, SystemMessage) for m in fake.seen)


async def test_summarizer_failure_falls_back_to_placeholder(monkeypatch):
    monkeypatch.setattr(summ, "build_chat_model", lambda cfg, role=None, **kw: _Boom())
    msgs = [
        HumanMessage(content="a"),
        AIMessage(content="b"),
        HumanMessage(content="c"),
        AIMessage(content="d"),
    ]
    result = (await compact({"messages": msgs}, _cfg(1, 0.0)))["messages"]
    assert isinstance(result[0], RemoveMessage)
    assert result[1].content == "[earlier conversation omitted]"
    assert [m.content for m in result[2:]] == ["d"]


def test_newest_durable_id_skips_synthetic_and_generated():
    head = [
        HumanMessage(content="s", id=None),
        HumanMessage(content="a", id="7"),
        AIMessage(content="b", id="lc_run-abc-0"),
        AIMessage(content="c", id="9"),
        AIMessage(content="d", id="lc_run-xyz-0"),
    ]
    assert _newest_durable_id(head) == "9"


def test_newest_durable_id_none_when_no_durable():
    head = [HumanMessage(content="s", id=None), AIMessage(content="x", id="lc_run-a-0")]
    assert _newest_durable_id(head) is None


async def test_compact_rides_watermark_on_summarizer_metadata(monkeypatch):
    fake = _CapturingSummarizer()
    monkeypatch.setattr(summ, "build_chat_model", lambda cfg, role=None, **kw: fake)
    msgs = [
        HumanMessage(content="a", id="1"),
        AIMessage(content="b", id="2"),
        HumanMessage(content="c", id="3"),
        AIMessage(content="d", id="4"),
    ]
    # budget 0 -> head is [1,2,3]; newest durable id folded is "3".
    await compact({"messages": msgs}, _cfg(1, 0.0))
    assert "summarization" in fake.config["tags"]
    assert fake.config["metadata"]["summary_through_id"] == "3"


def _tok(m):
    return count_tokens_approximately([m])


def test_split_keeps_largest_suffix_within_budget():
    msgs = [
        HumanMessage(content="a" * 400),
        AIMessage(content="b" * 400),
        HumanMessage(content="c" * 400),
        AIMessage(content="d" * 400),
    ]
    budget = _tok(msgs[2]) + _tok(msgs[3])
    head, tail = _split(msgs, budget)
    assert tail == msgs[2:]
    assert head == msgs[:2]


def test_split_always_keeps_newest_even_over_budget():
    msgs = [HumanMessage(content="a" * 400), AIMessage(content="z" * 4000)]
    head, tail = _split(msgs, 1)
    assert tail == [msgs[1]]
    assert head == [msgs[0]]


def test_split_snaps_tool_group_into_tail():
    ai = AIMessage(content="", tool_calls=[{"name": "t", "args": {}, "id": "1"}])
    tool = ToolMessage(content="r", tool_call_id="1")
    head, tail = _split([HumanMessage(content="q"), ai, tool], 1)
    assert [m.content for m in head] == ["q"]
    assert tail[0] is ai and tail[1] is tool
