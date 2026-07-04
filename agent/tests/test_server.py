import json

import httpx
import pytest
from langchain_core.messages import AIMessage, AIMessageChunk, HumanMessage


def _parse_sse(text):
    events = []
    for block in text.strip().split("\n\n"):
        if not block:
            continue
        ev, data = block.split("\n", 1)
        events.append(
            (ev.removeprefix("event: "), json.loads(data.removeprefix("data: ")))
        )
    return events


class _FakeAgent:
    """Stands in for the compiled graph: emits a scripted astream_events run."""

    async def astream_events(self, inp, version="v2", config=None):
        yield {
            "event": "on_chat_model_stream",
            "metadata": {"langgraph_node": "agent"},
            "data": {"chunk": AIMessageChunk(content="Hello")},
        }
        yield {
            "event": "on_chat_model_end",
            "data": {
                "output": AIMessage(
                    content="Hello",
                    usage_metadata={
                        "input_tokens": 3,
                        "output_tokens": 2,
                        "total_tokens": 5,
                        "output_token_details": {"reasoning": 0},
                    },
                )
            },
        }


class _RaisingAgent:
    """Fails partway through a run to exercise the error path."""

    async def astream_events(self, inp, version="v2", config=None):
        raise RuntimeError("upstream boom")
        yield  # unreachable; makes this an async generator


@pytest.fixture
def app(monkeypatch):
    from src import server

    monkeypatch.setattr(server, "agent", _FakeAgent())
    return server.app


@pytest.fixture
def raising_app(monkeypatch):
    from src import server

    monkeypatch.setattr(server, "agent", _RaisingAgent())
    return server.app


async def test_healthz(app):
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(transport=transport, base_url="http://t") as c:
        r = await c.get("/healthz")
        assert r.status_code == 200
        assert r.json() == {"status": "ok"}


async def test_run_streams_token_usage_end(app):
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(transport=transport, base_url="http://t") as c:
        r = await c.post("/run", json={"messages": [{"role": "user", "content": "hi"}]})
        assert r.status_code == 200
        events = _parse_sse(r.text)
        assert [e for e, _ in events] == ["token", "usage", "end"]
        assert events[0][1] == {"text": "Hello"}
        assert events[1][1]["total"] == 5


async def test_run_emits_error_then_end_on_failure(raising_app):
    transport = httpx.ASGITransport(app=raising_app)
    async with httpx.AsyncClient(transport=transport, base_url="http://t") as c:
        r = await c.post("/run", json={"messages": [{"role": "user", "content": "hi"}]})
        events = _parse_sse(r.text)
        assert [e for e, _ in events] == ["error", "end"]
        assert "boom" in events[0][1]["message"]


def test_to_lc_messages_maps_roles():
    from src import server

    msgs = server._to_lc_messages(
        [{"role": "user", "content": "u"}, {"role": "assistant", "content": "a"}]
    )
    assert isinstance(msgs[0], HumanMessage) and msgs[0].content == "u"
    assert isinstance(msgs[1], AIMessage) and msgs[1].content == "a"


def test_sse_framing():
    from src import server

    out = server._sse({"event": "token", "data": {"text": "hi"}})
    assert out == 'event: token\ndata: {"text": "hi"}\n\n'
