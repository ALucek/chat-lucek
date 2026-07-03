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

variable "billing_account" {
  type        = string
  description = "Billing account ID the project is linked to, for the budget."
}

variable "budget_amount" {
  type        = number
  description = "Monthly budget in USD; thresholds alert as spend crosses it."
  default     = 20
}