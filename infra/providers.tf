provider "google" {
  project = var.project_id
  region  = var.region

  # Billing budgets and some APIs require a quota project on the request.
  user_project_override = true
  billing_project       = var.project_id
}

# Online-eval auth; values come from terraform.tfvars.
provider "langsmith" {
  api_key      = var.langsmith_api_key
  workspace_id = var.langsmith_workspace_id
}