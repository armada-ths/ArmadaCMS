resource "google_recaptcha_enterprise_key" "website" {
  count = var.enable_recaptcha ? 1 : 0

  project      = var.project_id
  display_name = "${var.name_prefix}-website-key"
  labels       = local.labels

  web_settings {
    integration_type  = "SCORE"
    allow_all_domains = false
    allowed_domains   = var.recaptcha_allowed_domains
  }

  depends_on = [
    google_project_service.enabled["recaptchaenterprise.googleapis.com"],
  ]
}
