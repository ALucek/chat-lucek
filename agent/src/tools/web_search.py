from functools import lru_cache

from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field
from tavily import AsyncTavilyClient

MAX_RESULTS = 5
RELEVANCE_SCORE_THRESHOLD = 0.7
TOOL_DESCRIPTION = """Internet Search Tool, takes in a natural language query and returns back relevant results + snippets from the web.
Usage guidelines:
1. Use Natural Language - The search tool is designed to handle semantic queries, avoid search engine operators or specific syntax.
2. Be Specific - Avoid broad, multi-topic queries. Call this tool multiple times for different topics.
"""


class WebSearchInput(BaseModel):
    query: str = Field(description="Natural-language search query.")


@lru_cache(maxsize=1)
def _client() -> AsyncTavilyClient:
    return AsyncTavilyClient()


async def _search(query: str) -> str:
    results = await _client().search(
        query,
        max_results=MAX_RESULTS,
        auto_parameters=True,
        include_raw_content=False,
        include_answer=False,
    )
    return process_search_results(results)


def build_web_search_tool() -> StructuredTool:
    """Build the internet search tool; the model supplies only a query."""
    return StructuredTool.from_function(
        coroutine=_search,
        name="internet_search",
        description=TOOL_DESCRIPTION,
        args_schema=WebSearchInput,
    )


def process_search_results(results: dict) -> str:
    """Filter out low-relevance results and format the rest as markdown for LLM input."""
    items = [
        item
        for item in results.get("results", [])
        if item.get("score", 0) >= RELEVANCE_SCORE_THRESHOLD
    ]
    query = results.get("query", "")
    lines = [f"## Search results for: {query}\n"]

    for index, item in enumerate(items, start=1):
        title = item.get("title", "")
        url = item.get("url", "")
        lines.append(f"{index}. [{title}]({url})")
        content = item.get("content", "")
        if content:
            lines.append(f"\t- {content}\n")

    return "\n".join(lines)
