import pytest
from langchain_core.messages import HumanMessage
from langsmith import testing as t

from evals.harness import run_step, tool_names
from src.graphs.agent import agent

SUITE = "chat-lucek-routing"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_plans_multi_step_task():
    question = (
        "Research three competing electric-car battery chemistries, compare them, "
        "and write up a summary table."
    )
    t.log_inputs({"message": question})
    reply = await run_step(agent, {"messages": [HumanMessage(content=question)]})
    t.log_outputs({"tool_calls": sorted(tool_names(reply))})
    assert "set_todos" in tool_names(reply)


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_no_plan_for_trivial_request():
    question = "What does the acronym CPU stand for?"
    t.log_inputs({"message": question})
    reply = await run_step(agent, {"messages": [HumanMessage(content=question)]})
    t.log_outputs({"tool_calls": sorted(tool_names(reply))})
    assert "set_todos" not in tool_names(reply)


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_delegates_research_to_subagent():
    question = "What are reviewers currently saying about the newest iPhone?"
    t.log_inputs({"message": question})
    reply = await run_step(agent, {"messages": [HumanMessage(content=question)]})
    t.log_outputs({"tool_calls": sorted(tool_names(reply))})
    assert "run_subagent" in tool_names(reply)


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_answers_simple_question_directly():
    question = "What is 17 times 24?"
    t.log_inputs({"message": question})
    reply = await run_step(agent, {"messages": [HumanMessage(content=question)]})
    t.log_outputs({"tool_calls": sorted(tool_names(reply))})
    assert "run_subagent" not in tool_names(reply)
