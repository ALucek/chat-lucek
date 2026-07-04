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
