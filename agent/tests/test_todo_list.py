import json

import pytest
from pydantic import ValidationError

from src.tools.todo_list import TodoItem, build_todo_tool, _set_todos


def test_set_todos_returns_full_list_as_json():
    todos = [
        TodoItem(description="plan", progress="completed"),
        TodoItem(description="write", progress="in progress"),
    ]
    data = json.loads(_set_todos(todos))
    assert [t["description"] for t in data["todos"]] == ["plan", "write"]
    assert data["todos"][0]["progress"] == "completed"


def test_progress_must_be_a_known_status():
    with pytest.raises(ValidationError):
        TodoItem(description="x", progress="nope")


def test_build_todo_tool_naming():
    tool = build_todo_tool()
    assert tool.name == "set_todos"
