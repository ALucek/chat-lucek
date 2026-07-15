import logging

from langchain_core.messages import HumanMessage, RemoveMessage, ToolMessage
from langchain_core.messages.utils import count_tokens_approximately
from langchain_core.runnables import RunnableConfig
from langgraph.graph.message import REMOVE_ALL_MESSAGES

from src.config import AgentConfig
from src.prompts.summary_prompt import get_summary_prompt
from src.utils import build_chat_model

logger = logging.getLogger("chat-agent")

SUMMARY_PREFIX = "Here is a summary of the earlier conversation so far:\n\n"


def _tokens(msg) -> int:
    return count_tokens_approximately([msg])


def _split(messages: list, budget: int) -> tuple[list, list]:
    if not messages:
        return [], []
    cut = len(messages) - 1  # always keep the newest message
    used = _tokens(messages[cut])
    while cut > 0 and used + _tokens(messages[cut - 1]) <= budget:
        cut -= 1
        used += _tokens(messages[cut])
    # Never orphan a tool result: pull its AIMessage into the tail.
    while cut > 0 and isinstance(messages[cut], ToolMessage):
        cut -= 1
    return messages[:cut], messages[cut:]


def _newest_durable_id(head: list) -> str | None:
    for m in reversed(head):
        mid = getattr(m, "id", None)
        if isinstance(mid, str) and mid.isdigit():
            return mid
    return None


async def _summarize(
    head, cfg: AgentConfig, config: RunnableConfig, wid: str | None
) -> HumanMessage:
    model = build_chat_model(cfg, role="summarizer")
    prompt = [*head, HumanMessage(get_summary_prompt())]
    meta = {**((config or {}).get("metadata") or {}), "summary_through_id": wid}
    cfg_tagged = {**(config or {}), "tags": ["summarization"], "metadata": meta}
    resp = await model.ainvoke(prompt, config=cfg_tagged)
    return HumanMessage(SUMMARY_PREFIX + resp.text)


async def compact(state: dict, config: RunnableConfig) -> dict:
    messages = state["messages"]
    cfg = AgentConfig.from_runnable_config(config)
    if count_tokens_approximately(messages) < cfg.summary_threshold:
        return {}
    budget = int(cfg.summary_keep_ratio * cfg.summary_threshold)
    head, tail = _split(messages, budget)
    if not head:
        return {}
    wid = _newest_durable_id(head)
    try:
        kept = [await _summarize(head, cfg, config, wid), *tail]
    except Exception:  # noqa: BLE001
        logger.warning("summarization failed", exc_info=True)
        kept = [HumanMessage("[earlier conversation omitted]"), *tail]
    return {"messages": [RemoveMessage(id=REMOVE_ALL_MESSAGES), *kept]}
