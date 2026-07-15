import asyncio
import json
import logging
import sys

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
        assert events[0][1]["message"] == "The agent run failed. Please try again."
        assert "boom" not in events[0][1]["message"]


async def test_run_forwards_thread_id_to_run_config(monkeypatch):
    from src import server

    seen = {}

    class _CapturingAgent:
        async def astream_events(self, inp, version="v2", config=None):
            seen["config"] = config
            return
            yield  # unreachable; makes this an async generator

    monkeypatch.setattr(server, "agent", _CapturingAgent())
    transport = httpx.ASGITransport(app=server.app)
    async with httpx.AsyncClient(transport=transport, base_url="http://t") as c:
        await c.post(
            "/run",
            json={"messages": [{"role": "user", "content": "hi"}], "thread_id": "42"},
        )
    assert seen["config"]["metadata"] == {"thread_id": "42"}


async def test_run_failure_logs_at_error(raising_app):
    from src import server

    records = []

    class _Capture(logging.Handler):
        def emit(self, record):
            records.append(record)

    cap = _Capture()
    server.logger.addHandler(cap)
    try:
        transport = httpx.ASGITransport(app=raising_app)
        async with httpx.AsyncClient(transport=transport, base_url="http://t") as c:
            await c.post("/run", json={"messages": [{"role": "user", "content": "hi"}]})
    finally:
        server.logger.removeHandler(cap)
    # A swallowed run failure must still surface as an ERROR log for the metric.
    assert any(r.levelno == logging.ERROR for r in records)


def test_cloudrun_formatter_sets_severity_and_message():
    from src import server

    rec = logging.LogRecord("chat-agent", logging.ERROR, "f.py", 1, "boom", None, None)
    out = json.loads(server._CloudRunLogFormatter().format(rec))
    assert out["severity"] == "ERROR"
    assert out["message"] == "boom"


def test_cloudrun_formatter_includes_traceback():
    from src import server

    try:
        raise ValueError("kaboom")
    except ValueError:
        rec = logging.LogRecord(
            "chat-agent", logging.ERROR, "f.py", 1, "failed", None, sys.exc_info()
        )
    out = json.loads(server._CloudRunLogFormatter().format(rec))
    assert "kaboom" in out["message"] and "Traceback" in out["message"]


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


def test_make_tracer_dev_uses_dev_project(monkeypatch):
    from src import server

    captured = {}

    def fake_tracer(project_name=None):
        captured["project"] = project_name
        return "t"

    monkeypatch.setattr(server, "tracing_is_enabled", lambda: True)
    monkeypatch.setattr(server, "LangChainTracer", fake_tracer)
    monkeypatch.setenv("LANGSMITH_PROJECT_DEV", "chat-lucek-dev")

    assert server._make_tracer(True) == "t"
    assert captured["project"] == "chat-lucek-dev"

    captured.clear()
    assert server._make_tracer(False) == "t"
    assert captured["project"] is None


async def test_interrupt_closes_open_runs(monkeypatch):
    from src import server

    tracer = _FakeTracer(run_map={"open": _FakeRun(None)}, client=_FakeClient())
    monkeypatch.setattr(server, "agent", _FakeAgent())
    monkeypatch.setattr(server, "_make_tracer", lambda *_: tracer)

    resp = await server.run(_FakeReq({"messages": [{"role": "user", "content": "hi"}]}))
    agen = resp.body_iterator
    await agen.__anext__()  # start the run
    with pytest.raises(asyncio.CancelledError):
        await agen.athrow(asyncio.CancelledError())  # client disconnects mid-run
    assert {rid for rid, _ in tracer.client.updates} == {"open"}
