from src.tools.web_search import build_web_search_tool, process_search_results


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


def test_tool_exposes_only_query():
    tool = build_web_search_tool()
    assert list(tool.args.keys()) == ["query"]
