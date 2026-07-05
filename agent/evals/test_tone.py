import pytest
from langchain_core.messages import HumanMessage
from langsmith import testing as t

from evals.harness import judge
from src.config import build_run_config
from src.graphs.agent import agent

SUITE = "chat-lucek-tone"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_tone_is_plain_and_direct():
    question = (
        "hey can u explain how https actually works, like the whole handshake thing 😅"
    )
    t.log_inputs({"message": question})
    result = await agent.ainvoke(
        {"messages": [HumanMessage(content=question)]}, build_run_config()
    )
    answer = result["messages"][-1].content
    t.log_outputs({"answer": answer})

    verdict = judge(
        answer,
        "The reply speaks plainly and answers directly, getting to the point "
        "without padding.",
        key="tone_plain_and_direct",
    )
    assert verdict.passed, verdict.reasoning


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_tone_has_no_filler():
    question = (
        "heyyy good morning!! hope you're well — quick q, how do i boil an egg? 😊"
    )
    t.log_inputs({"message": question})
    result = await agent.ainvoke(
        {"messages": [HumanMessage(content=question)]}, build_run_config()
    )
    answer = result["messages"][-1].content
    t.log_outputs({"answer": answer})

    verdict = judge(
        answer,
        "The reply has no filler pleasantries, forced enthusiasm, verbose greeting "
        "or farewell fluff, or 'how can I help' loops.",
        key="tone_no_filler",
    )
    assert verdict.passed, verdict.reasoning


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_tone_does_not_perform():
    question = "ugh i'm honestly terrible at math, could u pleaseee help me understand compound interest?? 🥺"
    t.log_inputs({"message": question})
    result = await agent.ainvoke(
        {"messages": [HumanMessage(content=question)]}, build_run_config()
    )
    answer = result["messages"][-1].content
    t.log_outputs({"answer": answer})

    verdict = judge(
        answer,
        "The reply does not perform: no sycophancy, no overeagerness, no emoji "
        "spam, and no AI-isms or self-referential talk about being an AI.",
        key="tone_does_not_perform",
    )
    assert verdict.passed, verdict.reasoning
