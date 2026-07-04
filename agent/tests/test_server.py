import json

import httpx
import pytest
from langchain_core.messages import AIMessage, AIMessageChunk


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


@pytest.fixture
def app(monkeypatch):
    from src import server

    monkeypatch.setattr(server, "agent", _FakeAgent())
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
