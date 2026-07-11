# Eval alert triage

A PII or prompt-injection [alert](../monitoring.md#tracing) means an online evaluator scored a live trace positive. These are signals to investigate, not levers: read the trace, decide real versus false positive, and act only if warranted.

## Find the trace

In the LangSmith prod project, filter recent root runs by the feedback key from the email (`pii_detected = 1` or `prompt_injection_score = 1`) and open the flagged run.

## PII in an answer

The match matters when the PII is not public or readily accessible, which almost always means it came from the user or the conversation rather than a web search:

- Non-public PII from the user or context: the real signal. Sensitive data is now in the answer and the trace; delete the message if warranted (the terms already ask users not to submit it).
- Public, readily accessible info, such as a business contact surfaced from the web: low concern, no action.
- Regex noise, such as an IP in a technical answer or a fake example the model invented: tune [`pii_scan.js`](../../agent/evals/online/pii_scan.js) to cut the false positive.

## Prompt injection attempt

The judge flags the attempt, not whether it worked. Check what the agent did next:

- Held the line: internet noise, no action.
- Broke character: harden the system prompt against that pattern.
- Persistent or targeted from one source: climb the [abuse ladder](abuse.md) to block the IP or close signups.

## Negative feedback spike

Not an evaluator: this alert fires when 5 or more replies get a thumbs-down within an hour. It is a quality signal, so triage is about the product, not security.

Filter recent root runs by `user_score = 0` and read the flagged replies together. Look for a shared cause: a bad prompt change just shipped, a model or provider issue, a topic the agent handles poorly, or one frustrated user thumbing many replies. Read any `user_feedback` notes for the reason in the user's own words. The fix depends on the cause (prompt tuning, a rollback, or nothing if it is noise); there is no emergency lever.

## Notes

- No alert here has an emergency lever; the response is triage, prompt hardening, or the abuse ladder.