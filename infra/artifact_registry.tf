resource "google_artifact_registry_repository" "chat" {
  repository_id = "chat"
  location      = var.region
  format        = "DOCKER"
  description   = "Container images for the chat app (api + web)."

  depends_on = [google_project_service.enabled]
}