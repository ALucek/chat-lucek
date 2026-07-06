# Online evals: JS code evaluators scoring live prod traces via IaC.

data "langsmith_project" "prod" {
  name = var.langsmith_project
}

resource "langsmith_evaluator" "pii" {
  name = "pii-scan"
  type = "code"

  code_evaluator = {
    language = "javascript"
    code     = file("${path.module}/../agent/evals/online/pii_scan.js")
  }
}

resource "langsmith_evaluator" "overcap" {
  name = "overcapped-searches"
  type = "code"

  code_evaluator = {
    language = "javascript"
    code     = file("${path.module}/../agent/evals/online/overcapped_searches.js")
  }
}

# Scan every root (main-agent) answer for PII.
resource "langsmith_run_rule" "pii" {
  display_name  = "pii scan on answers"
  session_id    = data.langsmith_project.prod.id
  sampling_rate = 1
  filter        = "eq(is_root, true)"
  evaluator_id  = langsmith_evaluator.pii.id
}

# Count over-cap search calls on each subagent graph run.
resource "langsmith_run_rule" "overcap" {
  display_name  = "over-cap search count"
  session_id    = data.langsmith_project.prod.id
  sampling_rate = 1
  filter        = "and(eq(name, \"LangGraph\"), eq(metadata_key, \"langgraph_node\"), eq(metadata_value, \"subagent\"))"
  evaluator_id  = langsmith_evaluator.overcap.id
}

# Prompt-injection LLM judge on user inputs; model from the workspace default.
resource "langsmith_evaluator" "injection" {
  name = "prompt-injection-scan"
  type = "llm"

  llm_evaluator = {
    prompt_repo_handle      = "chat-lucek-prompt-injection"
    commit_hash_or_tag      = var.injection_prompt_commit
    variable_mapping_json   = jsonencode({ input = "input" })
    use_corrections_dataset = false
  }
}

# Judge 100% of root-run inputs (spend limits unavailable on this org plan).
resource "langsmith_run_rule" "injection" {
  display_name  = "prompt injection scan on inputs"
  session_id    = data.langsmith_project.prod.id
  sampling_rate = 1
  filter        = "eq(is_root, true)"
  evaluator_id  = langsmith_evaluator.injection.id
}

# Thread-level helpfulness judge; reads the whole assembled conversation.
resource "langsmith_evaluator" "helpfulness" {
  name = "thread-helpfulness"
  type = "llm"

  llm_evaluator = {
    prompt_repo_handle      = "chat-lucek-thread-helpfulness"
    commit_hash_or_tag      = var.thread_helpfulness_prompt_commit
    variable_mapping_json   = jsonencode({ outputs = "all_messages" })
    use_corrections_dataset = false
  }
}

# Judge 10% of completed threads (group_by thread_id makes it thread-level).
resource "langsmith_run_rule" "helpfulness" {
  display_name  = "thread helpfulness scan"
  session_id    = data.langsmith_project.prod.id
  sampling_rate = 0.1
  group_by      = "thread_id"
  evaluator_id  = langsmith_evaluator.helpfulness.id
}

resource "langsmith_evaluator" "conversation_length" {
  name = "conversation-length"
  type = "code"

  code_evaluator = {
    language = "javascript"
    code     = file("${path.module}/../agent/evals/online/conversation_length.js")
  }
}

# Count user turns on each root run; last value per thread is its length.
resource "langsmith_run_rule" "conversation_length" {
  display_name  = "conversation length on root runs"
  session_id    = data.langsmith_project.prod.id
  sampling_rate = 1
  filter        = "eq(is_root, true)"
  evaluator_id  = langsmith_evaluator.conversation_length.id
}

# Email alerts: page the owner when a security evaluator scores positive.
# LangSmith POSTs the static body to Resend's send API; the API key lives in
# config (TF state), so use a send-scoped key.
locals {
  resend_alert_headers = jsonencode({
    "Authorization" = "Bearer ${var.resend_api_key}"
    "Content-Type"  = "application/json"
  })

  project_url = "https://smith.langchain.com/o/${var.langsmith_workspace_id}/projects/p/${data.langsmith_project.prod.id}"

  pii_alert_html = <<-EOT
    <p>The pii evaluator flagged an assistant answer in a live conversation on ${var.langsmith_project}. The response matched a PII pattern such as an SSN, email address, phone number, card number, IP address, passport, or license number.</p>
    <p>Open the project in LangSmith and filter recent root runs by pii_detected = 1 to find the trace: <a href="${local.project_url}">${var.langsmith_project}</a></p>
    <p>This alert fires when the average pii_detected score rises above 0 in a 15 minute window.</p>
  EOT

  injection_alert_html = <<-EOT
    <p>The prompt-injection judge flagged a user's latest message as an injection or jailbreak attempt in a live conversation on ${var.langsmith_project}.</p>
    <p>Open the project in LangSmith and filter recent root runs by prompt_injection_score = 1 to review it: <a href="${local.project_url}">${var.langsmith_project}</a></p>
    <p>This alert fires when the average prompt_injection_score rises above 0 in a 15 minute window.</p>
  EOT
}

# Alert on any answer flagged by the pii evaluator.
resource "langsmith_alert_rule" "pii" {
  session_id     = data.langsmith_project.prod.id
  name           = "pii detected in answers"
  description    = "A prod answer scored positive on the pii evaluator."
  type           = "threshold"
  attribute      = "feedback_score"
  aggregation    = "avg"
  operator       = "gt"
  threshold      = 0
  window_minutes = 15
  filter         = "eq(feedback_key, \"pii_detected\")"

  actions = [{
    target = "webhook"
    config_json = jsonencode({
      project_name = var.langsmith_project
      url          = "https://api.resend.com/emails"
      headers      = local.resend_alert_headers
      body = jsonencode({
        from    = var.alert_email_from
        to      = [var.owner_email]
        subject = "[chat.lucek.ai] PII detected in a production answer"
        html    = local.pii_alert_html
      })
    })
  }]
}

# Alert on any user message flagged by the prompt-injection judge.
resource "langsmith_alert_rule" "injection" {
  session_id     = data.langsmith_project.prod.id
  name           = "prompt injection detected"
  description    = "A user message scored positive on the prompt-injection judge."
  type           = "threshold"
  attribute      = "feedback_score"
  aggregation    = "avg"
  operator       = "gt"
  threshold      = 0
  window_minutes = 15
  filter         = "eq(feedback_key, \"prompt_injection_score\")"

  actions = [{
    target = "webhook"
    config_json = jsonencode({
      project_name = var.langsmith_project
      url          = "https://api.resend.com/emails"
      headers      = local.resend_alert_headers
      body = jsonencode({
        from    = var.alert_email_from
        to      = [var.owner_email]
        subject = "[chat.lucek.ai] Prompt injection attempt detected"
        html    = local.injection_alert_html
      })
    })
  }]
}
