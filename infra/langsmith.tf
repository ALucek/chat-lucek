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
    prompt_repo_handle    = "chat-lucek-prompt-injection"
    commit_hash_or_tag    = var.injection_prompt_commit
    variable_mapping_json = jsonencode({ input = "input" })
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
