import importlib


def test_package_imports_without_credentials(monkeypatch):
    """The package must import with no secrets set, so CI, ruff, and type
    checkers (which all import modules) run without real keys. This is the
    durable guarantee behind lazy tool construction."""
    for key in ("OPENROUTER_API_KEY", "TAVILY_API_KEY", "LANGSMITH_API_KEY"):
        monkeypatch.delenv(key, raising=False)
    for mod in ("src.graphs.subagent", "src.graphs.agent"):
        importlib.reload(importlib.import_module(mod))
