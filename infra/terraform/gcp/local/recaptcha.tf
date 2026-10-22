# Dedicated developer keys, held in their own state. No hosted resources.
resource "google_recaptcha_enterprise_key" "local_test" {
  project      = var.project_id
  display_name = "armadacms-local-test-key"
  labels       = { app = "armadacms", environment = "local", managed-by = "terraform" }

  web_settings {
    integration_type = "SCORE"
    # Developer LAN addresses change. The local API still validates hostnames.
    allow_all_domains = true
  }

  testing_options {
    testing_score = 1.0
  }

}

resource "google_apikeys_key" "local_recaptcha_assessment" {
  project      = var.project_id
  name         = "armadacms-local-recaptcha-assessment"
  display_name = "Local development reCAPTCHA assessments"

  restrictions {
    api_targets {
      service = "recaptchaenterprise.googleapis.com"
    }
  }

}

output "local_recaptcha_config" {
  description = "Development-only configuration consumed by scripts/configure-local-recaptcha.ps1."
  sensitive   = true
  value = {
    project_id = var.project_id
    site_key   = reverse(split("/", google_recaptcha_enterprise_key.local_test.name))[0]
    api_key    = google_apikeys_key.local_recaptcha_assessment.key_string
  }
}
