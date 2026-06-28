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