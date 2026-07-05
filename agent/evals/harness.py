import json
import os
from pathlib import Path

from langchain_core.messages import (
    AIMessage,
    AnyMessage,
    HumanMessage,
    SystemMessage,
    ToolMessage,
)
from langchain_openrouter import ChatOpenRouter
from langsmith import testing as t
from pydantic import BaseModel, Field

from src.config import build_run_config

# Pinned judge model, separate from the agent. Override via EVAL_JUDGE_MODEL.
JUDGE_MODEL = "anthropic/claude-sonnet-5"

_FIXTURES = Path(__file__).parent / "fixtures"

_JUDGE_SYSTEM = (
    "You are a strict evaluator. For each assertion, first reason about whether "
    "the content satisfies it, then decide pass or fail. Judge only what is asked. "
    "If a reference is provided, treat it as one example of an ideal response, not "
    "the only acceptable one."
)


def judge_model_name() -> str:
    return os.getenv("EVAL_JUDGE_MODEL", JUDGE_MODEL)


class AssertResult(BaseModel):
    assertion: str = Field(description="The assertion being evaluated.")
    reasoning: str = Field(
        description="Reasoning about whether the content satisfies it."
    )
    passed: bool = Field(
        description="True only if the content satisfies the assertion."
    )


class JudgeResult(BaseModel):
    results: list[AssertResult]


def _to_message(m: dict) -> AnyMessage:
    role = m["role"]
    if role == "human":
        return HumanMessage(content=m["content"])
    if role == "ai":
        return AIMessage(
            content=m.get("content", ""), tool_calls=m.get("tool_calls", [])
        )
    if role == "tool":
        return ToolMessage(
            content=m["content"], name=m.get("name"), tool_call_id=m["tool_call_id"]
        )
    raise ValueError(f"unknown message role {role!r}")


def load_state(name: str) -> dict:
    """Load a JSON trace fixture into a graph state dict."""
    data = json.loads((_FIXTURES / name).read_text())
    state: dict = {"messages": [_to_message(m) for m in data["messages"]]}
    if "search_count" in data:
        state["search_count"] = data["search_count"]
    return state


async def run_step(graph, state: dict) -> AnyMessage:
    """Run a compiled graph a single node and return the message it produced."""
    stream = graph.astream(state, build_run_config(), stream_mode="updates")
    update = await stream.__anext__()
    await stream.aclose()
    return next(iter(update.values()))["messages"][-1]


def tool_names(message: AnyMessage) -> set[str]:
    return {call["name"] for call in getattr(message, "tool_calls", None) or []}


def judge(
    content: str,
    asserts: dict[str, str],
    *,
    prefix: str = "eval",
    reference: str | None = None,
) -> JudgeResult:
    """Grade content against named assertions (reason, then boolean, each).

    Logs one boolean feedback per assertion under "<prefix>_<name>". An optional
    reference is shown to the judge as one example of an ideal response.
    """
    model = ChatOpenRouter(
        model=judge_model_name(), temperature=0
    ).with_structured_output(JudgeResult)
    listed = "\n".join(f"- {a}" for a in asserts.values())
    blocks = [f"<assertions>\n{listed}\n</assertions>"]
    if reference is not None:
        blocks.append(f"<reference>\n{reference}\n</reference>")
    blocks.append(f"<content>\n{content}\n</content>")
    prompt = [
        SystemMessage(content=_JUDGE_SYSTEM),
        HumanMessage(content="\n\n".join(blocks)),
    ]
    with t.trace_feedback():
        result = model.invoke(prompt)
        for name, r in zip(asserts, result.results):
            t.log_feedback(key=f"{prefix}_{name}", score=r.passed)
    return result
