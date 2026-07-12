from langchain_core.messages import AnyMessage, SystemMessage
from langchain_openrouter import ChatOpenRouter

from src.config import AgentConfig, get_settings


def build_chat_model(config: AgentConfig, *, role: str) -> ChatOpenRouter:
    kwargs = {
        "streaming": True,
        "stream_usage": True,
        "max_retries": get_settings().model_max_retries,
        **config.chat_kwargs(role),
    }
    return ChatOpenRouter(**kwargs)


def build_messages(system_prompt: str, history: list[AnyMessage]) -> list[AnyMessage]:
    """Prepend the system prompt to the messages as given."""
    return [SystemMessage(content=system_prompt), *history]
