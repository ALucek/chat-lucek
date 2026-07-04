from langchain_core.messages import HumanMessage
from langchain_core.runnables import RunnableConfig
from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field

from src.config import get_settings
from src.graphs.subagent import subagent


class SubagentTask(BaseModel):
    task: str = Field(description="The task to delegate to the subagent.")


SUBAGENT_TOOL_DESCRIPTION = """Delegate a focused task to a subagent that works independently in its own context window and returns a summary.
Usage guidelines:
1. Specify Intent - The subagent is another LLM based agent that can be given further instructions or clarification to complete the task.
2. Be Specific - Subagents excel at individual, well-scoped tasks. Avoid broad, multi-part tasks. Call this tool multiple times for different tasks.
"""


async def _run_subagent(task: str, config: RunnableConfig) -> str:
    # Subagent gets its own recursion budget, not the parent's.
    sub_config = {**config, "recursion_limit": get_settings().subagent_recursion_limit}
    result = await subagent.ainvoke(
        {"messages": [HumanMessage(content=task)]},
        config=sub_config,
    )
    last_message = result["messages"][-1]
    return getattr(last_message, "content", "") or ""


def build_subagent_tool() -> StructuredTool:
    return StructuredTool.from_function(
        coroutine=_run_subagent,
        name="run_subagent",
        description=SUBAGENT_TOOL_DESCRIPTION,
        args_schema=SubagentTask,
    )
