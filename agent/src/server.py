import asyncio
import json
import logging
import os
import sys
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


class _CloudRunLogFormatter(logging.Formatter):
    # Emit JSON so Cloud Run reads severity; ERROR feeds the agent_errors metric.
    def format(self, record: logging.LogRecord) -> str:
        entry = {"severity": record.levelname, "message": record.getMessage()}
        if record.exc_info:
            entry["message"] += "\n" + self.formatException(record.exc_info)
        return json.dumps(entry)


def _make_logger() -> logging.Logger:
    log = logging.getLogger("chat-agent")
    if not log.handlers:
        handler = logging.StreamHandler(sys.stdout)
        handler.setFormatter(_CloudRunLogFormatter())
        log.addHandler(handler)
        log.setLevel(logging.INFO)
        log.propagate = False
    return log


logger = _make_logger()


@app.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


def _to_lc_messages(messages: list[dict[str, Any]]) -> list[Any]:
    out: list[Any] = []
    for m in messages:
        cls = HumanMessage if m.get("role") == "user" else AIMessage
        out.append(cls(content=m.get("content", ""), id=m.get("id")))
    return out


def _sse(event: dict[str, Any]) -> str:
    return f"event: {event['event']}\ndata: {json.dumps(event['data'])}\n\n"


def _make_tracer(dev: bool = False) -> LangChainTracer | None:
    # A tracer we own, so an interrupted run can close its runs. None if off.
    if not tracing_is_enabled():
        return None
    # Dev-host runs trace to the dev project, off the prod evaluators.
    if dev and (project := os.environ.get("LANGSMITH_PROJECT_DEV")):
        return LangChainTracer(project_name=project)
    return LangChainTracer()


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
                logger.warning("failed to close interrupted run %s", run_id)


@app.post("/run")
async def run(req: Request) -> StreamingResponse:
    body = await req.json()
    messages = _to_lc_messages(body.get("messages", []))
    config = build_run_config(body.get("overrides"), body.get("thread_id"))
    tracer = _make_tracer(bool(body.get("dev")))
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
            if tracer is not None:
                meta = tr.meta_event()
                if meta is not None:
                    yield _sse(meta)
        except (asyncio.CancelledError, GeneratorExit):
            if tracer is not None:
                _close_open_runs(tracer)
            raise
        except Exception:  # noqa: BLE001
            logger.exception("agent run failed")
            yield _sse(
                {
                    "event": "error",
                    "data": {"message": "The agent run failed. Please try again."},
                }
            )
        yield _sse({"event": "end", "data": {}})

    return StreamingResponse(gen(), media_type="text/event-stream")
