# Eval alert triage

A PII or prompt-injection [alert](../monitoring.md#tracing) means an online evaluator scored a live trace positive. These are signals to investigate, not levers: read the trace, decide real versus false positive, and act only if warranted.

## Find the trace

In the LangSmith prod project, filter recent root runs by the feedback key from the email (`pii_detected = 1` or `prompt_injection_score = 1`) and open the flagged run.

## PII in an answer

Read the answer and where the match came from:

- The user's own details echoed back, or a fake example the model invented: benign, no action.
- Real third-party PII from a web search: a privacy concern. Tighten the system prompt so it does not surface personal data, and delete the message if needed.
- Regex noise, such as an IP in a technical answer or a public contact address: tune [`pii_scan.js`](../../agent/evals/online/pii_scan.js) to cut the false positive.

## Prompt injection attempt

The judge flags the attempt, not whether it worked. Check what the agent did next:

- Held the line: internet noise, no action.
- Broke character, leaked its prompt, or did something it should not: harden the system prompt against that pattern.
- Persistent or targeted from one source: climb the [abuse ladder](abuse.md) to block the IP or close signups.

## Notes

- Neither alert has an emergency lever; the response is triage, prompt hardening, or the abuse ladder.
- Frequent false positives turn the alert into noise. Tune the evaluator in [`agent/evals/online/`](../../agent/evals/README.md) rather than ignoring the email.
