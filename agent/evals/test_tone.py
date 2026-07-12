import pytest

from evals.harness import answer_for, judge

SUITE = "chat-lucek-tone"


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_tone_is_plain_and_direct():
    answer = await answer_for(
        "hey can u explain how https actually works, like the whole handshake thing 😅"
    )
    verdict = judge(
        answer,
        "The reply speaks plainly and answers directly, getting to the point "
        "without padding.",
        key="tone_plain_and_direct",
    )
    assert verdict.passed, verdict.reasoning


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_tone_has_no_filler():
    answer = await answer_for(
        "heyyy good morning!! hope you're well — quick q, how do i boil an egg? 😊"
    )
    verdict = judge(
        answer,
        "The reply has no filler pleasantries, forced enthusiasm, verbose greeting "
        "or farewell fluff, or 'how can I help' loops.",
        key="tone_no_filler",
    )
    assert verdict.passed, verdict.reasoning


@pytest.mark.langsmith(test_suite_name=SUITE)
async def test_tone_does_not_perform():
    answer = await answer_for(
        "ugh i'm honestly terrible at math, could u pleaseee help me understand compound interest?? 🥺"
    )
    verdict = judge(
        answer,
        "The reply does not perform: no sycophancy, no overeagerness, no emoji "
        "spam, and no AI-isms or self-referential talk about being an AI.",
        key="tone_does_not_perform",
    )
    assert verdict.passed, verdict.reasoning
