resource "google_compute_global_address" "lb" {
  name = "chat-lb-ip"
}

resource "google_compute_region_network_endpoint_group" "api" {
  name                  = "chat-api-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.api.name
  }
}

resource "google_compute_region_network_endpoint_group" "web" {
  name                  = "chat-web-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.web.name
  }
}

resource "google_compute_backend_service" "api" {
  name                  = "chat-api-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.api.id
  backend {
    group = google_compute_region_network_endpoint_group.api.id
  }
}

resource "google_compute_backend_service" "web" {
  name                  = "chat-web-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  backend {
    group = google_compute_region_network_endpoint_group.web.id
  }
}

# Candidate NEGs route to the "cand"-tagged revision, served on the dev host.
resource "google_compute_region_network_endpoint_group" "api_cand" {
  name                  = "chat-api-cand-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.api.name
    tag     = "cand"
  }
}

resource "google_compute_region_network_endpoint_group" "web_cand" {
  name                  = "chat-web-cand-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_v2_service.web.name
    tag     = "cand"
  }
}

# Candidate backends are IAP-gated; only owner + deploy SA reach dev.
resource "google_compute_backend_service" "api_cand" {
  name                  = "chat-api-cand-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.api.id
  backend {
    group = google_compute_region_network_endpoint_group.api_cand.id
  }
  iap {
    enabled              = true
    oauth2_client_id     = var.iap_oauth_client_id
    oauth2_client_secret = var.iap_oauth_client_secret
  }
}

resource "google_compute_backend_service" "web_cand" {
  name                  = "chat-web-cand-backend"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  backend {
    group = google_compute_region_network_endpoint_group.web_cand.id
  }
  iap {
    enabled              = true
    oauth2_client_id     = var.iap_oauth_client_id
    oauth2_client_secret = var.iap_oauth_client_secret
  }
}

resource "google_compute_url_map" "lb" {
  name            = "chat-url-map"
  default_service = google_compute_backend_service.web.id

  host_rule {
    hosts        = [var.domain]
    path_matcher = "main"
  }

  path_matcher {
    name            = "main"
    default_service = google_compute_backend_service.web.id

    path_rule {
      paths   = ["/api", "/api/*"]
      service = google_compute_backend_service.api.id
    }

    # api+db health, for the external uptime check
    path_rule {
      paths   = ["/readyz"]
      service = google_compute_backend_service.api.id
    }
  }

  # Private dev host: same routing as prod, but to the candidate backends.
  host_rule {
    hosts        = [var.dev_domain]
    path_matcher = "dev"
  }

  path_matcher {
    name            = "dev"
    default_service = google_compute_backend_service.web_cand.id

    path_rule {
      paths   = ["/api", "/api/*"]
      service = google_compute_backend_service.api_cand.id
    }

    # /agentz is the dev-only api-to-agent integration check.
    path_rule {
      paths   = ["/readyz", "/agentz"]
      service = google_compute_backend_service.api_cand.id
    }
  }
}

# Only the owner and deploy SA pass through IAP to the dev host.
resource "google_iap_web_backend_service_iam_member" "dev_api_owner" {
  web_backend_service = google_compute_backend_service.api_cand.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = "user:${var.owner_email}"
}

resource "google_iap_web_backend_service_iam_member" "dev_web_owner" {
  web_backend_service = google_compute_backend_service.web_cand.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = "user:${var.owner_email}"
}

resource "google_iap_web_backend_service_iam_member" "dev_api_deploy" {
  web_backend_service = google_compute_backend_service.api_cand.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = "serviceAccount:${google_service_account.deploy.email}"
}

resource "google_iap_web_backend_service_iam_member" "dev_web_deploy" {
  web_backend_service = google_compute_backend_service.web_cand.name
  role                = "roles/iap.httpsResourceAccessor"
  member              = "serviceAccount:${google_service_account.deploy.email}"
}

resource "google_compute_managed_ssl_certificate" "cert" {
  name = "chat-cert"
  managed {
    domains = [var.domain]
  }
}

# Separate dev cert so an unvalidated dev domain can't stall the prod cert.
resource "google_compute_managed_ssl_certificate" "dev_cert" {
  name = "chat-dev-cert"
  managed {
    domains = [var.dev_domain]
  }
}

resource "google_compute_target_https_proxy" "https" {
  name             = "chat-https-proxy"
  url_map          = google_compute_url_map.lb.id
  ssl_certificates = [google_compute_managed_ssl_certificate.cert.id, google_compute_managed_ssl_certificate.dev_cert.id]
}

resource "google_compute_global_forwarding_rule" "https" {
  name                  = "chat-https-fr"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "443"
  ip_address            = google_compute_global_address.lb.address
  target                = google_compute_target_https_proxy.https.id
}

resource "google_compute_url_map" "redirect" {
  name = "chat-redirect"
  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "http" {
  name    = "chat-http-proxy"
  url_map = google_compute_url_map.redirect.id
}

resource "google_compute_global_forwarding_rule" "http" {
  name                  = "chat-http-fr"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "80"
  ip_address            = google_compute_global_address.lb.address
  target                = google_compute_target_http_proxy.http.id
}