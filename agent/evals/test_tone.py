import pytest
from langchain_core.messages import HumanMessage
from langsmith import testing as t

from evals.harness import judge
from src.config import build_run_config
from src.graphs.agent import agent

SUITE = "chat-lucek-tone"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_answer_follows_the_prompt_tone():
    # Deliberately casual, emoji-laden prompt to tempt a matching reply.
    # Asserts are grounded in the agent prompt's <tone>/<behavior> sections.
    question = "hey!! can u gimme a quick fun rundown of how espresso differs from regular coffee?? 😄"
    t.log_inputs({"message": question})
    result = await agent.ainvoke(
        {"messages": [HumanMessage(content=question)]}, build_run_config()
    )
    answer = result["messages"][-1].content
    t.log_outputs({"answer": answer})

    verdict = judge(
        answer,
        {
            "plain_and_direct": (
                "The reply speaks plainly and answers directly, getting to the "
                "point without padding."
            ),
            "no_filler": (
                "The reply has no filler pleasantries, forced enthusiasm, verbose greeting "
                "or farewell fluff, or 'how can I help' loops."
            ),
            "no_performing": (
                "The reply does not perform: no sycophancy, no overeagerness, no "
                "emoji spam, and no AI-isms or self-referential talk about being an AI."
            ),
        },
        prefix="tone",
    )
    failed = [r for r in verdict.results if not r.passed]
    assert not failed, [f"{r.assertion}: {r.reasoning}" for r in failed]
