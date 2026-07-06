# Thread-level helpfulness LLM judge
from langchain_core.prompts.structured import StructuredPrompt
from langchain_openai import ChatOpenAI
from langsmith import Client

HANDLE = "chat-lucek-thread-helpfulness"

SYSTEM = (
    "You are a strict evaluator of a chat assistant's helpfulness across an "
    "entire conversation. Read the whole thread and decide whether the assistant "
    "was genuinely helpful from the user's point of view: it understood what the "
    "user wanted and effectively resolved it by the end. Judge the conversation "
    "as a whole, not any single turn. Signals that it was NOT helpful: the user "
    "had to repeat or rephrase the same request, the user had to correct or "
    "re-steer the assistant repeatedly, the user expressed frustration, doubt, or "
    "dissatisfaction, or the user's goal was left unresolved. A short "
    "conversation that simply answers the user well is helpful."
)

SCHEMA = {
    "title": "thread_helpfulness",
    "description": "Whether the assistant was helpful across the conversation.",
    "type": "object",
    "properties": {
        "thread_helpful": {
            "type": "boolean",
            "description": "True if the assistant was helpful across the "
            "conversation, else False.",
        },
        "thread_helpful_explanation": {
            "type": "string",
            "description": "One sentence explaining the verdict.",
        },
    },
    "required": ["thread_helpful", "thread_helpful_explanation"],
}


def build():
    prompt = StructuredPrompt(
        [
            ("system", SYSTEM),
            (
                "human",
                "Grade this conversation:\n\n<conversation>\n{all_messages}\n</conversation>",
            ),
        ],
        SCHEMA,
        structured_output_kwargs={"strict": True},
    )
    # OpenRouter via OpenAI-compatible base; secret ref is OPENAI_API_KEY.
    model = ChatOpenAI(
        model="openai/gpt-5.4-mini",
        base_url="https://openrouter.ai/api/v1",
        temperature=0,
    )
    return prompt | model


def main() -> None:
    client = Client()
    url = client.push_prompt(HANDLE, object=build())
    print("prompt url:", url)
    print("commit_hash:", client.pull_prompt_commit(HANDLE).commit_hash)


if __name__ == "__main__":
    main()
