import pytest

from evals.harness import answer_for, judge

SUITE = "chat-lucek-ability"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_knows_its_identity():
    answer = await answer_for("Who are you?")
    assert "harold" in answer.lower(), answer


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_completes_a_web_search():
    # Full run: delegates to the subagent, searches the web, and answers.
    answer = await answer_for("What's the weather in NYC right now?")
    verdict = judge(
        answer,
        "Pass only if the reply answers with concrete, current information that "
        "could only come from a live web search (here, NYC's present weather "
        "with real details). Fail if it declines, errors out, says it cannot "
        "search or reach the web, or gives only generic guidance.",
        key="completes_web_search",
    )
    assert verdict.passed, verdict.reasoning
