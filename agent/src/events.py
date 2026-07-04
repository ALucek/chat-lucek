from typing import Any


def _serialize_output(out: Any) -> Any:
    content = getattr(out, "content", None)
    if content is not None:
        return content
    if isinstance(out, (str, int, float, bool, list, dict)) or out is None:
        return out
    return str(out)


class Translator:
    """Translates astream_events v2 into ordered node/delta/node_end frames."""

    def __init__(self) -> None:
        self._open_tools: set[str] = set()
        self._started: set[str] = set()
        self._input = self._output = self._total = self._reasoning = 0

    def handle(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        etype = event.get("event")
        if etype == "on_chat_model_stream":
            return self._on_stream(event)
        if etype == "on_tool_start":
            return self._on_tool_start(event)
        if etype == "on_tool_end":
            return self._on_tool_end(event)
        if etype == "on_chat_model_end":
            self._accumulate(event)
        return []

    def _parent_id(self, event: dict[str, Any]) -> str | None:
        for pid in reversed(event.get("parent_ids") or []):
            if str(pid) in self._open_tools:
                return str(pid)
        return None

    def _text_frames(
        self, node_id: str, parent: str | None, ntype: str, text: str
    ) -> list[dict[str, Any]]:
        if node_id not in self._started:
            self._started.add(node_id)
            return [
                {
                    "event": "node",
                    "data": {"id": node_id, "parent_id": parent, "type": ntype},
                },
                {"event": "delta", "data": {"id": node_id, "text": text}},
            ]
        return [{"event": "delta", "data": {"id": node_id, "text": text}}]

    def _on_stream(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        chunk = (event.get("data") or {}).get("chunk")
        parent = self._parent_id(event)
        rid = str(event.get("run_id"))
        out: list[dict[str, Any]] = []
        reasoning = (getattr(chunk, "additional_kwargs", {}) or {}).get(
            "reasoning_content"
        )
        if reasoning:
            out += self._text_frames(f"{rid}:reasoning", parent, "reasoning", reasoning)
        content = getattr(chunk, "content", "")
        if content:
            out += self._text_frames(f"{rid}:text", parent, "text", content)
        return out

    def _on_tool_start(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        rid = str(event.get("run_id"))
        parent = self._parent_id(event)
        self._open_tools.add(rid)
        self._started.add(rid)
        inp = (event.get("data") or {}).get("input") or {}
        return [
            {
                "event": "node",
                "data": {
                    "id": rid,
                    "parent_id": parent,
                    "type": "tool",
                    "name": event.get("name", ""),
                    "input": inp,
                },
            }
        ]

    def _on_tool_end(self, event: dict[str, Any]) -> list[dict[str, Any]]:
        rid = str(event.get("run_id"))
        self._open_tools.discard(rid)
        out = (event.get("data") or {}).get("output")
        return [
            {"event": "node_end", "data": {"id": rid, "output": _serialize_output(out)}}
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
