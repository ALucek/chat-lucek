locals {
  secret_ids = ["jwt-secret", "openrouter-api-key", "db-password", "google-client-secret", "tavily-api-key", "langsmith-api-key", "usage-hash-secret", "resend-api-key"]
}

resource "google_secret_manager_secret" "app" {
  for_each  = toset(local.secret_ids)
  secret_id = each.value

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

# API Resend key mirrors the LangSmith alert key; single tfvar to rotate.
resource "google_secret_manager_secret_version" "resend_api_key" {
  secret      = google_secret_manager_secret.app["resend-api-key"].id
  secret_data = var.resend_api_key
}