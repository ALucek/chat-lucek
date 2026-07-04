from langchain_core.messages import AIMessage, AIMessageChunk

from src.events import Translator


def _stream(node, content="", reasoning=None):
    chunk = AIMessageChunk(
        content=content,
        additional_kwargs={"reasoning_content": reasoning} if reasoning else {},
    )
    return {
        "event": "on_chat_model_stream",
        "metadata": {"langgraph_node": node},
        "data": {"chunk": chunk},
    }


def _tool(name, state, run_id, tool_input=None):
    ev = f"on_tool_{state}"
    data = {"input": tool_input} if tool_input is not None else {}
    return {"event": ev, "name": name, "run_id": run_id, "data": data}


def _model_end(total, reasoning=0):
    msg = AIMessage(
        content="",
        usage_metadata={
            "input_tokens": 1,
            "output_tokens": total - 1,
            "total_tokens": total,
            "output_token_details": {"reasoning": reasoning},
        },
    )
    return {"event": "on_chat_model_end", "data": {"output": msg}}


def test_answer_tokens_only_from_top_level():
    t = Translator()
    assert t.handle(_stream("agent", content="Hi")) == [
        {"event": "token", "data": {"text": "Hi"}}
    ]
    # subagent content is internal, not surfaced
    assert t.handle(_stream("subagent", content="internal")) == []


def test_reasoning_surfaced_from_top_level():
    t = Translator()
    assert t.handle(_stream("agent", reasoning="thinking")) == [
        {"event": "reasoning", "data": {"text": "thinking"}}
    ]
    assert t.handle(_stream("subagent", reasoning="hidden")) == []


def test_status_search_with_query():
    t = Translator()
    out = t.handle(_tool("internet_search", "start", "r1", {"query": "cats"}))
    assert out == [
        {
            "event": "status",
            "data": {"id": "r1", "kind": "search", "detail": "cats", "state": "start"},
        }
    ]


def test_status_research_with_task_and_parallel():
    t = Translator()
    a = t.handle(_tool("run_subagent", "start", "a", {"task": "A"}))
    b = t.handle(_tool("run_subagent", "start", "b", {"task": "B"}))
    assert a[0]["data"]["id"] == "a" and a[0]["data"]["detail"] == "A"
    assert b[0]["data"]["id"] == "b" and b[0]["data"]["detail"] == "B"
    end_a = t.handle(_tool("run_subagent", "end", "a"))
    assert end_a == [
        {
            "event": "status",
            "data": {"id": "a", "kind": "research", "detail": "", "state": "end"},
        }
    ]


def test_status_plan_for_set_todos():
    out = Translator().handle(_tool("set_todos", "start", "p1", {"todos": []}))
    assert out == [
        {
            "event": "status",
            "data": {"id": "p1", "kind": "plan", "detail": "", "state": "start"},
        }
    ]


def test_unknown_tool_ignored():
    assert Translator().handle(_tool("mystery", "start", "x", {})) == []


def test_usage_accumulates_across_calls():
    t = Translator()
    t.handle(_model_end(total=10, reasoning=3))
    t.handle(_model_end(total=5, reasoning=1))
    assert t.usage_event() == {
        "event": "usage",
        "data": {"input": 2, "output": 13, "total": 15, "reasoning": 4},
    }
