# Publish one online-eval prompt to LangSmith Prompt Hub.
import importlib.util
import sys
from pathlib import Path

from langsmith import Client


def load(path: str):
    spec = importlib.util.spec_from_file_location(Path(path).stem, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: push_prompt.py <path-to-prompt.py>")
    module = load(sys.argv[1])
    client = Client()
    url = client.push_prompt(module.HANDLE, object=module.build())
    print("prompt url:", url)
    commit = client.pull_prompt_commit(module.HANDLE)
    print("commit_hash:", commit.commit_hash)


if __name__ == "__main__":
    main()
