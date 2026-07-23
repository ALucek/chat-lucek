locals {
  # Only whether the URL is set is needed here, not its (sensitive) value.
  upstash_enabled = nonsensitive(var.upstash_redis_url != "")
  secret_ids = concat(
    ["jwt-secret", "openrouter-api-key", "db-password", "google-client-secret", "tavily-api-key", "langsmith-api-key", "usage-hash-secret", "resend-api-key"],
    local.upstash_enabled ? ["upstash-redis-url"] : [],
  )
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

# Only seeded when the tfvar is set; the API stays in-memory otherwise.
resource "google_secret_manager_secret_version" "upstash_redis_url" {
  count       = local.upstash_enabled ? 1 : 0
  secret      = google_secret_manager_secret.app["upstash-redis-url"].id
  secret_data = var.upstash_redis_url
}