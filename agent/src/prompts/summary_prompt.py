def get_summary_prompt() -> str:
    return """Please write a summary of the transcript. 
The purpose of this summary is to provide continuity so you can continue to make progress towards solving the task in a future context, where the raw history above may not be accessible and will be replaced with this summary.
If a summary of earlier turns is already present, incorporate it and produce a single updated summary rather than starting over.
Write down anything that would be helpful, including the state, next steps, learnings etc."""
