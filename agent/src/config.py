from functools import lru_cache
from typing import Any, ClassVar

from dotenv import load_dotenv
from pydantic import BaseModel, ConfigDict, Field
from pydantic_settings import BaseSettings, SettingsConfigDict

load_dotenv()


class Settings(BaseSettings):
    """Process config from the environment."""

    model_config = SettingsConfigDict(extra="ignore")

    default_model: str = "deepseek/deepseek-v4-flash"
    max_searches: int = 5
    max_tokens: int = 8192
    recursion_limit: int = 100
    subagent_recursion_limit: int = 50
    model_max_retries: int = 3
    summary_threshold: int = 100_000
    summary_keep_ratio: float = 0.1


@lru_cache
def get_settings() -> Settings:
    return Settings()


class RoleConfig(BaseModel):
    """Per-role overrides for model settings."""

    model_config = ConfigDict(extra="allow")
    model: str | None = Field(default=None, description="Model override for this role.")


class AgentConfig(BaseModel):
    """Configuration loaded from RunnableConfig.configurable."""

    model_config = ConfigDict(extra="ignore")

    agent: RoleConfig = Field(
        default_factory=RoleConfig, description="Overrides for the top-level agent."
    )
    subagent: RoleConfig = Field(
        default_factory=RoleConfig, description="Overrides for the subagent."
    )
    summarizer: RoleConfig = Field(
        default_factory=RoleConfig, description="Overrides for the summarizer."
    )
    max_searches: int = Field(
        default_factory=lambda: get_settings().max_searches,
        description="Hard limit on the number of web searches per subagent run.",
    )
    summary_threshold: int = Field(
        default_factory=lambda: get_settings().summary_threshold,
        description="Approx-token context size that triggers compaction.",
    )
    summary_keep_ratio: float = Field(
        default_factory=lambda: get_settings().summary_keep_ratio,
        description="Fraction of summary_threshold of recent messages kept verbatim.",
    )

    ROLES: ClassVar[set[str]] = {"agent", "subagent", "summarizer"}

    def chat_kwargs(self, role: str) -> dict[str, Any]:
        if role not in self.ROLES:
            raise ValueError(f"Unknown role {role!r}, expected one of {self.ROLES}")
        settings = get_settings()
        base = {"model": settings.default_model, "max_tokens": settings.max_tokens}
        return {**base, **getattr(self, role).model_dump(exclude_none=True)}

    @classmethod
    def from_runnable_config(cls, config: dict[str, Any] | None) -> "AgentConfig":
        configurable = (config or {}).get("configurable")
        if not isinstance(configurable, dict) or not configurable:
            return cls()
        return cls.model_validate(configurable)


def build_run_config(
    overrides: dict[str, Any] | None = None, thread_id: str | None = None
) -> dict[str, Any]:
    """Build a RunnableConfig from optional per-run overrides."""
    overrides = overrides or {}
    settings = get_settings()
    role: dict[str, Any] = {}
    if "model" in overrides:
        role["model"] = overrides["model"]
    if "max_tokens" in overrides:
        role["max_tokens"] = overrides["max_tokens"]
    configurable: dict[str, Any] = {
        "max_searches": overrides.get("max_searches", settings.max_searches),
        "summary_threshold": overrides.get(
            "summary_threshold", settings.summary_threshold
        ),
        "summary_keep_ratio": overrides.get(
            "summary_keep_ratio", settings.summary_keep_ratio
        ),
    }
    if role:
        configurable["agent"] = dict(role)
        configurable["subagent"] = dict(role)
    config: dict[str, Any] = {
        "recursion_limit": settings.recursion_limit,
        "configurable": configurable,
    }
    # thread_id groups this conversation's runs into one LangSmith thread.
    if thread_id:
        config["metadata"] = {"thread_id": thread_id}
    return config
