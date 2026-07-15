from types import SimpleNamespace

from src.events import Translator


def _chunk(content="", reasoning=""):
    kw = {"reasoning_content": reasoning} if reasoning else {}
    return SimpleNamespace(content=content, additional_kwargs=kw)


def _stream(run_id, parent_ids, chunk):
    return {
        "event": "on_chat_model_stream",
        "name": "ChatOpenRouter",
        "run_id": run_id,
        "parent_ids": parent_ids,
        "data": {"chunk": chunk},
    }


def _stream_tagged(run_id, parent_ids, chunk, tags):
    return {
        "event": "on_chat_model_stream",
        "name": "ChatOpenRouter",
        "run_id": run_id,
        "parent_ids": parent_ids,
        "tags": tags,
        "data": {"chunk": chunk},
    }


def test_summarization_chunks_emit_compaction_not_text():
    tr = Translator()
    out = tr.handle(
        _stream_tagged("s1", ["root"], _chunk(content="sum"), ["summarization"])
    )
    assert out == [
        {
            "event": "node",
            "data": {"id": "s1:compaction", "parent_id": None, "type": "compaction"},
        },
        {"event": "delta", "data": {"id": "s1:compaction", "text": "sum"}},
    ]


def test_first_reasoning_chunk_opens_node_then_delta():
    tr = Translator()
    out = tr.handle(_stream("r1", ["root", "n1"], _chunk(reasoning="Hi")))
    assert out == [
        {
            "event": "node",
            "data": {"id": "r1:reasoning", "parent_id": None, "type": "reasoning"},
        },
        {"event": "delta", "data": {"id": "r1:reasoning", "text": "Hi"}},
    ]


def test_second_chunk_same_kind_is_delta_only():
    tr = Translator()
    tr.handle(_stream("r1", ["root"], _chunk(reasoning="Hi")))
    out = tr.handle(_stream("r1", ["root"], _chunk(reasoning=" there")))
    assert out == [{"event": "delta", "data": {"id": "r1:reasoning", "text": " there"}}]


def test_reasoning_then_text_same_run_are_two_nodes():
    tr = Translator()
    tr.handle(_stream("r1", ["root"], _chunk(reasoning="think")))
    out = tr.handle(_stream("r1", ["root"], _chunk(content="answer")))
    assert out[0] == {
        "event": "node",
        "data": {"id": "r1:text", "parent_id": None, "type": "text"},
    }
    assert out[1] == {"event": "delta", "data": {"id": "r1:text", "text": "answer"}}


def test_empty_chunk_emits_nothing():
    tr = Translator()
    assert tr.handle(_stream("r1", ["root"], _chunk())) == []


def _tool_start(run_id, parent_ids, name, inp):
    return {
        "event": "on_tool_start",
        "name": name,
        "run_id": run_id,
        "parent_ids": parent_ids,
        "data": {"input": inp},
    }


def _tool_end(run_id, parent_ids, name, output):
    return {
        "event": "on_tool_end",
        "name": name,
        "run_id": run_id,
        "parent_ids": parent_ids,
        "data": {"output": output},
    }


def test_leaf_tool_start_and_end():
    tr = Translator()
    start = tr.handle(
        _tool_start("t1", ["root", "n"], "internet_search", {"query": "x"})
    )
    assert start == [
        {
            "event": "node",
            "data": {
                "id": "t1",
                "parent_id": None,
                "type": "tool",
                "name": "internet_search",
                "input": {"query": "x"},
            },
        }
    ]
    end = tr.handle(_tool_end("t1", ["root", "n"], "internet_search", {"results": []}))
    assert end == [
        {"event": "node_end", "data": {"id": "t1", "output": {"results": []}}}
    ]


def test_events_inside_subagent_nest_under_the_tool():
    tr = Translator()
    tr.handle(_tool_start("SA", ["root", "n"], "run_subagent", {"task": "research"}))
    sub_search = tr.handle(
        _tool_start("s1", ["root", "n", "SA", "sn"], "internet_search", {"query": "q"})
    )
    assert sub_search[0]["data"]["parent_id"] == "SA"
    text = tr.handle(
        _stream("m1", ["root", "n", "SA", "sn"], _chunk(content="summary"))
    )
    assert text[0]["data"]["parent_id"] == "SA"
    tr.handle(_tool_end("SA", ["root", "n"], "run_subagent", "done"))
    top = tr.handle(_stream("m2", ["root", "n2"], _chunk(content="final")))
    assert top[0]["data"]["parent_id"] is None


def test_tool_message_output_uses_content():
    tr = Translator()
    tr.handle(_tool_start("SA", ["root"], "run_subagent", {"task": "t"}))
    msg = SimpleNamespace(content="the summary")
    out = tr.handle(_tool_end("SA", ["root"], "run_subagent", msg))
    assert out == [{"event": "node_end", "data": {"id": "SA", "output": "the summary"}}]


def test_captures_root_run_id_and_builds_meta_event():
    tr = Translator()
    # The root event has empty parent_ids; its run_id is the trace root.
    tr.handle(_stream("root-run", [], _chunk(content="hi")))
    assert tr.meta_event() == {
        "event": "meta",
        "data": {"langsmith_run_id": "root-run"},
    }


def test_meta_event_is_none_without_a_root_event():
    tr = Translator()
    # Only nested events (non-empty parent_ids) -> no root captured.
    tr.handle(_stream("r1", ["root", "n"], _chunk(content="hi")))
    assert tr.meta_event() is None


def test_usage_tally_across_two_model_runs():
    tr = Translator()
    for tok in (10, 5):
        msg = SimpleNamespace(
            usage_metadata={
                "input_tokens": tok,
                "output_tokens": tok,
                "total_tokens": 2 * tok,
                "output_token_details": {"reasoning": 1},
            }
        )
        tr.handle({"event": "on_chat_model_end", "data": {"output": msg}})
    assert tr.usage_event()["data"] == {
        "input": 15,
        "output": 15,
        "total": 30,
        "reasoning": 2,
    }
