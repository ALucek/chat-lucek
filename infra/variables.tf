variable "project_id" {
  type        = string
  description = "GCP project hosting the deployment."
}

variable "region" {
  type        = string
  description = "Region for regional resources."
  default     = "us-central1"
}

variable "db_tier" {
  type        = string
  description = "Cloud SQL machine tier"
  default     = "db-f1-micro"
}

variable "google_client_id" {
  type        = string
  description = "Google OAuth client ID the API verifies tokens against."
}

variable "owner_email" {
  type        = string
  description = "Account granted the unlimited token budget."
}

variable "domain" {
  type        = string
  description = "Public domain served by the load balancer."
  default     = "chat.lucek.ai"
}

variable "github_repo" {
  type        = string
  description = "GitHub repo (owner/name) allowed to deploy via WIF."
  default     = "ALucek/chat-lucek"
}

variable "signup_open" {
  type        = bool
  description = "Whether new-user registration is allowed. Set false to close signups."
  default     = true
}

variable "magic_link_from" {
  type        = string
  description = "Verified Resend sender address for magic-link sign-in emails."
  default     = "login@lucek.ai"
}

variable "billing_account" {
  type        = string
  description = "Billing account ID the project is linked to, for the budget."
}

variable "budget_amount" {
  type        = number
  description = "Monthly budget in USD; thresholds alert as spend crosses it."
  default     = 20
}

variable "agent_default_model" {
  type        = string
  description = "Default model the agent uses, overridable per run."
  default     = "deepseek/deepseek-v4-flash"
}

variable "agent_max_searches" {
  type        = number
  description = "Max web searches per agent run."
  default     = 5
}

variable "langsmith_project" {
  type        = string
  description = "LangSmith project the agent traces into."
  default     = "simple-deep-research"
}

variable "langsmith_api_key" {
  type        = string
  description = "LangSmith API key for provisioning online evaluators."
  sensitive   = true
}

variable "langsmith_workspace_id" {
  type        = string
  description = "LangSmith workspace (tenant) ID that owns the evaluators."
}

variable "injection_prompt_commit" {
  type        = string
  description = "Pinned Prompt Hub commit hash for the prompt-injection judge."
}

variable "thread_helpfulness_prompt_commit" {
  type        = string
  description = "Pinned Prompt Hub commit hash for the thread-helpfulness judge."
}

variable "resend_api_key" {
  type        = string
  description = "Resend API key (send-scoped) for delivering online-eval alerts."
  sensitive   = true
}

variable "alert_email_from" {
  type        = string
  description = "From address for alert emails; must be a verified Resend sender."
  default     = "alerts@lucek.ai"
}