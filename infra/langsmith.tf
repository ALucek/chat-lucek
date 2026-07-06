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
