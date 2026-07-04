from typing import Any

TOP_LEVEL_NODE = "agent"
STATUS_KINDS = {
    "run_subagent": "research",
    "internet_search": "search",
    "set_todos": "plan",
}


def _detail(name: str, tool_input: dict[str, Any]) -> str:
    if name == "run_subagent":
        return str(tool_input.get("task", ""))
    if name == "internet_search":
        return str(tool_input.get("query", ""))
    return ""


class Translator:
    """Maps LangGraph astream_events to our SSE event dicts; tallies usage."""

    def __init__(self) -> None:
        self._input = 0
        self._output = 0
        self._total = 0
        self._reasoning = 0

    def handle(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        etype = event.get("event")
        if etype == "on_chat_model_stream":
            return self._on_stream(event)
        if etype in ("on_tool_start", "on_tool_end"):
            return self._on_tool(event)
        if etype == "on_chat_model_end":
            self._accumulate(event)
        return []

    def _on_stream(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        if (event.get("metadata") or {}).get("langgraph_node") != TOP_LEVEL_NODE:
            return []
        chunk = (event.get("data") or {}).get("chunk")
        out: list[dict[str, Any]] = []
        reasoning = (getattr(chunk, "additional_kwargs", {}) or {}).get(
            "reasoning_content"
        )
        if reasoning:
            out.append({"event": "reasoning", "data": {"text": reasoning}})
        if getattr(chunk, "content", ""):
            out.append({"event": "token", "data": {"text": chunk.content}})
        return out

    def _on_tool(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        kind = STATUS_KINDS.get(event.get("name", ""))
        if kind is None:
            return []
        state = "start" if event.get("event") == "on_tool_start" else "end"
        detail = ""
        if state == "start":
            detail = _detail(
                event["name"], (event.get("data") or {}).get("input") or {}
            )
        return [
            {
                "event": "status",
                "data": {
                    "id": str(event.get("run_id")),
                    "kind": kind,
                    "detail": detail,
                    "state": state,
                },
            }
        ]

    def _accumulate(self, event: dict[str, Any]) -> None:
        msg = (event.get("data") or {}).get("output")
        um = getattr(msg, "usage_metadata", None) or {}
        self._input += um.get("input_tokens", 0)
        self._output += um.get("output_tokens", 0)
        self._total += um.get("total_tokens", 0)
        self._reasoning += (um.get("output_token_details") or {}).get("reasoning", 0)

    def usage_event(self) -> dict[str, Any]:
        return {
            "event": "usage",
            "data": {
                "input": self._input,
                "output": self._output,
                "total": self._total,
                "reasoning": self._reasoning,
            },
        }
