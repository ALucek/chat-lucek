import pytest
from langchain_core.messages import HumanMessage
from langsmith import testing as t

from evals.harness import judge
from src.config import build_run_config
from src.graphs.agent import agent

SUITE = "chat-lucek-ability"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_knows_its_identity():
    question = "Who are you?"
    t.log_inputs({"message": question})
    result = await agent.ainvoke(
        {"messages": [HumanMessage(content=question)]}, build_run_config()
    )
    answer = result["messages"][-1].content
    t.log_outputs({"answer": answer})
    assert "harold" in answer.lower(), answer


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_completes_a_web_search():
    # Full run: delegates to the subagent, searches the web, and answers.
    question = "What's the weather in NYC right now?"
    t.log_inputs({"message": question})
    result = await agent.ainvoke(
        {"messages": [HumanMessage(content=question)]}, build_run_config()
    )
    answer = result["messages"][-1].content
    t.log_outputs({"answer": answer})

    verdict = judge(
        answer,
        "Pass only if the reply answers with concrete, current information that "
        "could only come from a live web search (here, NYC's present weather "
        "with real details). Fail if it declines, errors out, says it cannot "
        "search or reach the web, or gives only generic guidance.",
        key="completes_web_search",
    )
    assert verdict.passed, verdict.reasoning
