import asyncio
import json
from datetime import datetime, timezone
from typing import Any

from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse
from langchain_core.messages import AIMessage, HumanMessage
from langchain_core.tracers.langchain import LangChainTracer
from langsmith.utils import tracing_is_enabled

from src.config import build_run_config
from src.events import Translator
from src.graphs.agent import agent

app = FastAPI()


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


def _to_lc_messages(messages: list[dict[str, Any]]) -> list[Any]:
    out: list[Any] = []
    for m in messages:
        if m.get("role") == "user":
            out.append(HumanMessage(content=m.get("content", "")))
        else:
            out.append(AIMessage(content=m.get("content", "")))
    return out


def _sse(event: dict[str, Any]) -> str:
    return f"event: {event['event']}\ndata: {json.dumps(event['data'])}\n\n"


def _make_tracer() -> LangChainTracer | None:
    # A tracer we own, so an interrupted run can close its runs. None if off.
    return LangChainTracer() if tracing_is_enabled() else None


def _close_open_runs(tracer: LangChainTracer) -> None:
    # Interrupt strands in-flight nested runs; close them so the trace ends.
    now = datetime.now(timezone.utc)
    for run_id, run in list(tracer.run_map.items()):
        if run.end_time is None:
            try:
                tracer.client.update_run(
                    run_id, end_time=now, error="Client interrupted"
                )
            except Exception:  # noqa: BLE001
                pass


@app.post("/run")
async def run(req: Request) -> StreamingResponse:
    body = await req.json()
    messages = _to_lc_messages(body.get("messages", []))
    config = build_run_config(body.get("overrides"))
    tracer = _make_tracer()
    if tracer is not None:
        config = {**config, "callbacks": [tracer]}

    async def gen():
        tr = Translator()
        try:
            async for event in agent.astream_events(
                {"messages": messages}, version="v2", config=config
            ):
                for out in tr.handle(event):
                    yield _sse(out)
            yield _sse(tr.usage_event())
        except (asyncio.CancelledError, GeneratorExit):
            if tracer is not None:
                _close_open_runs(tracer)
            raise
        except Exception as exc:  # noqa: BLE001
            yield _sse({"event": "error", "data": {"message": str(exc)}})
        yield _sse({"event": "end", "data": {}})

    return StreamingResponse(gen(), media_type="text/event-stream")
