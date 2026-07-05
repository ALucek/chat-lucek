import pytest

from src.config import AgentConfig, build_run_config, get_settings
from src.utils import build_chat_model


def test_chat_kwargs_rejects_unknown_role():
    with pytest.raises(ValueError, match="Unknown role"):
        AgentConfig().chat_kwargs("nonsense")


def test_settings_defaults(monkeypatch):
    for k in (
        "DEFAULT_MODEL",
        "MAX_SEARCHES",
        "MAX_TOKENS",
        "RECURSION_LIMIT",
        "SUBAGENT_RECURSION_LIMIT",
        "MODEL_MAX_RETRIES",
    ):
        monkeypatch.delenv(k, raising=False)
    get_settings.cache_clear()
    s = get_settings()
    assert s.default_model == "deepseek/deepseek-v4-flash"
    assert (s.max_searches, s.max_tokens) == (5, 8192)
    assert (s.recursion_limit, s.subagent_recursion_limit) == (100, 50)
    assert s.model_max_retries == 3


def test_settings_env_override(monkeypatch):
    monkeypatch.setenv("MAX_SEARCHES", "9")
    get_settings.cache_clear()
    assert get_settings().max_searches == 9
    get_settings.cache_clear()


def test_build_run_config_defaults():
    cfg = build_run_config({})
    assert cfg["recursion_limit"] == get_settings().recursion_limit
    assert cfg["configurable"]["max_searches"] == get_settings().max_searches
    assert "agent" not in cfg["configurable"]  # no per-role block without overrides


def test_build_run_config_overrides_apply_to_both_roles():
    cfg = build_run_config({"model": "x/y", "max_searches": 2, "max_tokens": 111})
    conf = cfg["configurable"]
    assert conf["max_searches"] == 2
    assert conf["agent"] == {"model": "x/y", "max_tokens": 111}
    assert conf["subagent"] == {"model": "x/y", "max_tokens": 111}


def test_build_run_config_sets_thread_id_metadata():
    cfg = build_run_config({}, thread_id="42")
    assert cfg["metadata"] == {"thread_id": "42"}


def test_build_run_config_omits_metadata_without_thread_id():
    assert "metadata" not in build_run_config({})


def test_chat_model_streams_with_default_max_tokens(monkeypatch):
    monkeypatch.setenv("OPENROUTER_API_KEY", "test")
    model = build_chat_model(AgentConfig(), role="agent")
    assert model.streaming is True
    assert model.stream_usage is True
    assert model.max_tokens == get_settings().max_tokens
    assert model.max_retries == get_settings().model_max_retries


def test_role_override_beats_default_max_tokens(monkeypatch):
    monkeypatch.setenv("OPENROUTER_API_KEY", "test")
    cfg = AgentConfig.from_runnable_config(
        {"configurable": {"agent": {"max_tokens": 256}}}
    )
    model = build_chat_model(cfg, role="agent")
    assert model.max_tokens == 256
