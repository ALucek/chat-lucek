import asyncio
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
            "run_id": "r1",
            "parent_ids": ["root", "n"],
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


async def test_run_streams_node_usage_end(app):
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(transport=transport, base_url="http://t") as c:
        r = await c.post("/run", json={"messages": [{"role": "user", "content": "hi"}]})
        assert r.status_code == 200
        events = _parse_sse(r.text)
        assert [e for e, _ in events] == ["node", "delta", "usage", "end"]
        assert events[0][1] == {"id": "r1:text", "parent_id": None, "type": "text"}
        assert events[1][1] == {"id": "r1:text", "text": "Hello"}
        assert events[2][1]["total"] == 5


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

    out = server._sse({"event": "delta", "data": {"id": "r1:text", "text": "hi"}})
    assert out == 'event: delta\ndata: {"id": "r1:text", "text": "hi"}\n\n'


class _FakeRun:
    def __init__(self, end_time=None):
        self.end_time = end_time


class _FakeClient:
    def __init__(self, fail_ids=()):
        self.updates = []
        self.fail_ids = set(fail_ids)

    def update_run(self, run_id, **kwargs):
        if run_id in self.fail_ids:
            raise RuntimeError("patch failed")
        self.updates.append((run_id, kwargs))


class _FakeTracer:
    def __init__(self, run_map, client):
        self.run_map = run_map
        self.client = client


class _FakeReq:
    def __init__(self, body):
        self._body = body

    async def json(self):
        return self._body


def test_close_open_runs_closes_only_open_ones():
    from src import server

    client = _FakeClient()
    tracer = _FakeTracer(
        run_map={
            "open1": _FakeRun(end_time=None),
            "done": _FakeRun(end_time="already"),
            "open2": _FakeRun(end_time=None),
        },
        client=client,
    )
    server._close_open_runs(tracer)
    assert {rid for rid, _ in client.updates} == {"open1", "open2"}
    for _, kw in client.updates:
        assert kw["error"] == "Client interrupted"
        assert kw["end_time"] is not None


def test_close_open_runs_swallows_client_errors():
    from src import server

    client = _FakeClient(fail_ids={"open1"})
    tracer = _FakeTracer(
        run_map={"open1": _FakeRun(None), "open2": _FakeRun(None)}, client=client
    )
    server._close_open_runs(tracer)  # a failed patch must not abort the rest
    assert {rid for rid, _ in client.updates} == {"open2"}


def test_make_tracer_gated_on_tracing(monkeypatch):
    from src import server

    monkeypatch.setattr(server, "tracing_is_enabled", lambda: False)
    assert server._make_tracer() is None

    sentinel = object()
    monkeypatch.setattr(server, "tracing_is_enabled", lambda: True)
    monkeypatch.setattr(server, "LangChainTracer", lambda: sentinel)
    assert server._make_tracer() is sentinel


async def test_interrupt_closes_open_runs(monkeypatch):
    from src import server

    tracer = _FakeTracer(run_map={"open": _FakeRun(None)}, client=_FakeClient())
    monkeypatch.setattr(server, "agent", _FakeAgent())
    monkeypatch.setattr(server, "_make_tracer", lambda: tracer)

    resp = await server.run(_FakeReq({"messages": [{"role": "user", "content": "hi"}]}))
    agen = resp.body_iterator
    await agen.__anext__()  # start the run
    with pytest.raises(asyncio.CancelledError):
        await agen.athrow(asyncio.CancelledError())  # client disconnects mid-run
    assert {rid for rid, _ in tracer.client.updates} == {"open"}
