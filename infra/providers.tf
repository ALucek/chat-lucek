provider "google" {
  project = var.project_id
  region  = var.region

  # Billing budgets and some APIs require a quota project on the request.
  user_project_override = true
  billing_project       = var.project_id
}