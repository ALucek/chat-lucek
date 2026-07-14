from src.tools import web_search as ws
from src.tools.web_search import build_web_search_tool, process_search_results


class _FakeClient:
    def __init__(self):
        self.calls = []

    async def search(self, query, **kwargs):
        self.calls.append((query, kwargs))
        return {
            "query": query,
            "results": [{"title": "T", "url": "u", "content": "c", "score": 0.9}],
        }


def test_filters_below_relevance_threshold():
    results = {
        "query": "cats",
        "results": [
            {"title": "Good", "url": "http://g", "content": "gc", "score": 0.9},
            {"title": "Bad", "url": "http://b", "content": "bc", "score": 0.3},
        ],
    }
    out = process_search_results(results)
    assert "Search results for: cats" in out
    assert "Good" in out and "http://g" in out
    assert "Bad" not in out


def test_formats_title_url_and_content():
    results = {
        "query": "q",
        "results": [
            {"title": "T", "url": "http://u", "content": "snippet", "score": 1.0}
        ],
    }
    out = process_search_results(results)
    assert "[T](http://u)" in out
    assert "snippet" in out


def test_empty_results_still_has_header():
    out = process_search_results({"query": "x", "results": []})
    assert "Search results for: x" in out


def test_build_web_search_tool_naming():
    tool = build_web_search_tool()
    assert tool.name == "internet_search"
    assert "Internet Search" in tool.description


def test_tool_exposes_only_query():
    tool = build_web_search_tool()
    assert list(tool.args.keys()) == ["query"]


async def test_search_sends_only_query_with_fixed_params(monkeypatch):
    fake = _FakeClient()
    monkeypatch.setattr(ws, "_client", lambda: fake)
    out = await ws._search("cats")
    query, kwargs = fake.calls[0]
    assert query == "cats"
    assert kwargs == {
        "max_results": ws.MAX_RESULTS,
        "auto_parameters": True,
        "include_raw_content": False,
        "include_answer": False,
    }
    assert "[T](u)" in out
