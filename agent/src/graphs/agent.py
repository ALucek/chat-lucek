from langchain_core.messages import AIMessage
from langchain_core.runnables import RunnableConfig
from langgraph.graph import END, START, StateGraph
from langgraph.prebuilt import ToolNode

from src.config import AgentConfig, get_settings
from src.prompts.agent_prompt import get_agent_system_prompt
from src.state import AgentState
from src.tools.subagent_tool import build_subagent_tool
from src.tools.todo_list import build_todo_tool
from src.utils import build_chat_model, build_messages


tools = [build_subagent_tool(), build_todo_tool()]
tool_node = ToolNode(tools)


async def agent_node(state: AgentState, config: RunnableConfig) -> dict:
    cfg = AgentConfig.from_runnable_config(config)
    model = build_chat_model(cfg, role="agent").bind_tools(tools)
    messages = build_messages(get_agent_system_prompt(), state["messages"])
    response = await model.ainvoke(messages, config=config)
    return {"messages": [response]}


def route_agent(state: AgentState) -> str:
    last_message = state["messages"][-1]
    if isinstance(last_message, AIMessage) and last_message.tool_calls:
        return "tools"
    return "end"


builder = StateGraph(AgentState)
builder.add_node("agent", agent_node)
builder.add_node("tools", tool_node)

builder.add_edge(START, "agent")
builder.add_conditional_edges("agent", route_agent, {"tools": "tools", "end": END})
builder.add_edge("tools", "agent")

agent = builder.compile().with_config(recursion_limit=get_settings().recursion_limit)
