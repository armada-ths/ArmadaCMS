resource "google_compute_region_network_endpoint_group" "cloud_run_lb_neg" {
  count = var.enable_https_load_balancer ? 1 : 0

  project               = var.project_id
  name                  = local.lb_neg_name
  region                = var.region
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = var.service_name
  }
}

resource "google_compute_backend_service" "cloud_run_lb" {
  count = var.enable_https_load_balancer ? 1 : 0

  project                         = var.project_id
  name                            = local.lb_backend_name
  protocol                        = "HTTPS"
  port_name                       = "http"
  timeout_sec                     = 30
  load_balancing_scheme           = "EXTERNAL_MANAGED"
  connection_draining_timeout_sec = 0

  log_config {
    enable        = true
    sample_rate   = 1.0
    optional_mode = "EXCLUDE_ALL_OPTIONAL"
  }

  backend {
    group = google_compute_region_network_endpoint_group.cloud_run_lb_neg[0].id
  }
}

resource "google_compute_url_map" "cloud_run_lb" {
  count = var.enable_https_load_balancer ? 1 : 0

  project         = var.project_id
  name            = local.lb_url_map_name
  default_service = google_compute_backend_service.cloud_run_lb[0].id
}

resource "google_compute_managed_ssl_certificate" "cloud_run_lb" {
  count = var.enable_https_load_balancer && length(var.lb_managed_certificate_domains) > 0 ? 1 : 0

  project = var.project_id
  name    = local.lb_certificate_name

  managed {
    domains = var.lb_managed_certificate_domains
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_target_https_proxy" "cloud_run_lb" {
  count = var.enable_https_load_balancer ? 1 : 0

  project = var.project_id
  name    = local.lb_target_https_proxy_name
  url_map = google_compute_url_map.cloud_run_lb[0].id

  ssl_certificates = concat(
    var.lb_existing_ssl_certificate_ids,
    google_compute_managed_ssl_certificate.cloud_run_lb[*].id,
  )

  lifecycle {
    precondition {
      condition     = length(concat(var.lb_existing_ssl_certificate_ids, google_compute_managed_ssl_certificate.cloud_run_lb[*].id)) > 0
      error_message = "When enable_https_load_balancer is true, provide at least one certificate via lb_managed_certificate_domains or lb_existing_ssl_certificate_ids."
    }
  }
}

resource "google_compute_global_address" "cloud_run_lb" {
  count = var.enable_https_load_balancer && var.lb_reserve_global_ip ? 1 : 0

  project = var.project_id
  name    = local.lb_global_address_name
}

resource "google_compute_global_forwarding_rule" "cloud_run_lb_https" {
  count = var.enable_https_load_balancer ? 1 : 0

  project               = var.project_id
  name                  = local.lb_https_forwarding_rule_name
  target                = google_compute_target_https_proxy.cloud_run_lb[0].id
  port_range            = "443-443"
  ip_address            = var.lb_reserve_global_ip ? google_compute_global_address.cloud_run_lb[0].address : null
  load_balancing_scheme = "EXTERNAL_MANAGED"
}

resource "google_compute_url_map" "cloud_run_lb_redirect" {
  count = var.enable_https_load_balancer && var.lb_enable_http_redirect ? 1 : 0

  project = var.project_id
  name    = local.lb_redirect_url_map_name

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }

  lifecycle {
    ignore_changes = [description]
  }
}

resource "google_compute_target_http_proxy" "cloud_run_lb_redirect" {
  count = var.enable_https_load_balancer && var.lb_enable_http_redirect ? 1 : 0

  project = var.project_id
  name    = local.lb_target_http_proxy_name
  url_map = google_compute_url_map.cloud_run_lb_redirect[0].id
}

resource "google_compute_global_forwarding_rule" "cloud_run_lb_http" {
  count = var.enable_https_load_balancer && var.lb_enable_http_redirect ? 1 : 0

  project               = var.project_id
  name                  = local.lb_http_forwarding_rule_name
  target                = google_compute_target_http_proxy.cloud_run_lb_redirect[0].id
  port_range            = "80-80"
  ip_address            = var.lb_reserve_global_ip ? google_compute_global_address.cloud_run_lb[0].address : null
  load_balancing_scheme = "EXTERNAL_MANAGED"
}
