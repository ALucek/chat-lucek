import pytest

from evals.harness import judge_model_name
from src.config import get_settings


@pytest.fixture(scope="session")
def langsmith_experiment_metadata():
    return {
        "agent_model": get_settings().default_model,
        "judge_model": judge_model_name(),
    }
