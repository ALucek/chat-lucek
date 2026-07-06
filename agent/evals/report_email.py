"""Email the weekly eval run output to the owner via Resend."""

import html
import json
import os
import re
import sys
import urllib.request

ANSI = re.compile(r"\x1b\[[0-9;]*m")
OUTCOME = {"success": "passed", "failure": "failed"}


def main() -> None:
    log = ANSI.sub("", open(sys.argv[1], encoding="utf-8").read())

    raw = os.environ.get("EVAL_OUTCOME", "completed")
    outcome = OUTCOME.get(raw, raw)

    link = "https://smith.langchain.com"
    workspace = os.environ.get("LANGSMITH_WORKSPACE_ID", "")
    if workspace:
        link = f"{link}/o/{workspace}"

    body = {
        "from": os.environ.get("EVAL_REPORT_FROM", "alerts@lucek.ai"),
        "to": [os.environ["EVAL_REPORT_TO"]],
        "subject": f"[chat.lucek.ai] Weekly agent evals: {outcome}",
        "html": (
            f"<p>Weekly agent eval run ({outcome}). "
            f'History in <a href="{link}">LangSmith</a>.</p>'
            f"<pre>{html.escape(log)}</pre>"
        ),
    }
    req = urllib.request.Request(
        "https://api.resend.com/emails",
        data=json.dumps(body).encode(),
        headers={
            "Authorization": f"Bearer {os.environ['RESEND_API_KEY']}",
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(req) as resp:
        resp.read()
    print("report emailed")


if __name__ == "__main__":
    main()
