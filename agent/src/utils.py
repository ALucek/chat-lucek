from typing import Any

from langchain_core.messages import AnyMessage, SystemMessage
from langchain_openrouter import ChatOpenRouter

from src.config import AgentConfig


def build_chat_model(
    config: AgentConfig,
    *,
    role: str,
    **overrides: Any,
) -> ChatOpenRouter:
    kwargs = {"streaming": True, "stream_usage": True, **config.chat_kwargs(role)}
    if overrides:
        kwargs.update(overrides)
    return ChatOpenRouter(**kwargs)


def build_messages(system_prompt: str, history: list[AnyMessage]) -> list[AnyMessage]:
    """Prepend the system prompt to the messages as given."""
    return [SystemMessage(content=system_prompt), *history]
