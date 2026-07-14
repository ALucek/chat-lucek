import json

from src.tools.todo_list import TodoItem, _set_todos


def test_set_todos_returns_full_list_as_json():
    todos = [
        TodoItem(description="plan", progress="completed"),
        TodoItem(description="write", progress="in progress"),
    ]
    data = json.loads(_set_todos(todos))
    assert [t["description"] for t in data["todos"]] == ["plan", "write"]
    assert data["todos"][0]["progress"] == "completed"
