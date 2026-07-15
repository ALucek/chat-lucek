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


def _split(messages: list, keep: int) -> tuple[list, list]:
    cut = max(len(messages) - keep, 0)
    head, tail = messages[:cut], messages[cut:]
    while head and tail and isinstance(tail[0], ToolMessage):
        head, tail = head[:-1], [head[-1], *tail]
    return head, tail


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
    head, tail = _split(messages, cfg.summary_keep)
    if not head:
        return {}
    wid = _newest_durable_id(head)
    try:
        kept = [await _summarize(head, cfg, config, wid), *tail]
    except Exception:  # noqa: BLE001
        logger.warning("summarization failed", exc_info=True)
        kept = [HumanMessage("[earlier conversation omitted]"), *tail]
    return {"messages": [RemoveMessage(id=REMOVE_ALL_MESSAGES), *kept]}
