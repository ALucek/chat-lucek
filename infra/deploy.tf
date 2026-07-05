# Keyless GitHub Actions -> GCP auth (Workload Identity Federation).
resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-actions"
  display_name              = "GitHub Actions"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"

  attribute_mapping = {
    "google.subject"        = "assertion.sub"
    "attribute.repository"  = "assertion.repository"
    "attribute.environment" = "assertion.environment"
  }

  # Only tokens minted for this repo are accepted.
  attribute_condition = "assertion.repository == \"${var.github_repo}\""

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# Identity that CI impersonates. Holds no DB access by design.
resource "google_service_account" "deploy" {
  account_id   = "chat-deploy"
  display_name = "chat CI deploy"
}

# Only the repo's production environment may impersonate the deploy SA.
resource "google_service_account_iam_member" "deploy_wif" {
  service_account_id = google_service_account.deploy.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.environment/production"
}

resource "google_project_iam_member" "deploy_artifact_registry" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.deploy.email}"
}

resource "google_project_iam_member" "deploy_run" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.deploy.email}"
}

# Deploy revisions/jobs that run as the runtime SAs.
resource "google_service_account_iam_member" "deploy_actas_api" {
  service_account_id = google_service_account.api.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deploy.email}"
}

resource "google_service_account_iam_member" "deploy_actas_web" {
  service_account_id = google_service_account.web.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deploy.email}"
}

resource "google_service_account_iam_member" "deploy_actas_agent" {
  service_account_id = google_service_account.agent.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deploy.email}"
}

# Lets the rotate-jwt workflow add a new jwt-secret version.
resource "google_secret_manager_secret_iam_member" "deploy_jwt_version_adder" {
  secret_id = google_secret_manager_secret.app["jwt-secret"].secret_id
  role      = "roles/secretmanager.secretVersionAdder"
  member    = "serviceAccount:${google_service_account.deploy.email}"
}

# Least-privilege role to toggle the full-kill deny rule on the Armor policy.
resource "google_project_iam_custom_role" "armor_rule_editor" {
  role_id     = "armorRuleEditor"
  title       = "Cloud Armor Rule Editor"
  description = "Mutate rules on Cloud Armor security policies (full-kill toggle)."
  permissions = [
    "compute.securityPolicies.get",
    "compute.securityPolicies.update",
  ]
}

resource "google_project_iam_member" "deploy_armor" {
  project = var.project_id
  role    = google_project_iam_custom_role.armor_rule_editor.id
  member  = "serviceAccount:${google_service_account.deploy.email}"
}
