output "instance_connection_name" {
  value       = google_sql_database_instance.chat.connection_name
  description = "Cloud SQL connection name for the Cloud Run socket DSN."
}

output "instance_name" {
  value       = google_sql_database_instance.chat.name
  description = "Cloud SQL instance name."
}

output "api_url" {
  value       = google_cloud_run_v2_service.api.uri
  description = "Cloud Run URL of the API service."
}

output "web_url" {
  value       = google_cloud_run_v2_service.web.uri
  description = "Cloud Run URL of the web service."
}

output "lb_ip" {
  value       = google_compute_global_address.lb.address
  description = "Static IP for the chat.lucek.ai DNS A-record."
}

output "wif_provider" {
  value       = google_iam_workload_identity_pool_provider.github.name
  description = "WIF provider resource name for the GitHub Actions auth step."
}

output "deploy_sa_email" {
  value       = google_service_account.deploy.email
  description = "Deploy service account email CI impersonates."
}