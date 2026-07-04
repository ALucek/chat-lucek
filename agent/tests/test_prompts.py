from src.prompts.agent_prompt import get_agent_system_prompt
from src.prompts.subagent_prompt import get_subagent_system_prompt


def test_agent_prompt_names_its_tools():
    prompt = get_agent_system_prompt()
    assert "run_subagent" in prompt
    assert "set_todos" in prompt


def test_subagent_prompt_names_search_tool():
    assert "internet_search" in get_subagent_system_prompt()
