# App runtime secrets — prefixed so they don't collide with production secrets
# in the same GCP project.

resource "google_secret_manager_secret" "app" {
  for_each = local.secret_env_vars

  project   = var.project_id
  secret_id = each.value
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

# Secret *values* are set directly in GCP Secret Manager (console or gcloud), NOT via Terraform.
# This resource only fires when secret_values is non-empty, which it intentionally never is.
# Keeping it avoids a manual import step if the escape hatch is ever needed.
resource "google_secret_manager_secret_version" "app" {
  for_each = local.secret_value_keys

  secret      = google_secret_manager_secret.app[each.key].id
  secret_data = var.secret_values[each.key]
}

resource "google_secret_manager_secret_iam_member" "runtime_secret_access" {
  for_each = var.manage_secret_accessor_bindings ? local.secret_env_vars : {}

  project   = var.project_id
  secret_id = google_secret_manager_secret.app[each.key].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)}"
}

# GitHub App private key — used by cloudbuild.yaml for deployment tracking.
# manage_github_app_secret = false (default) reuses the secret already created
# by the production workspace, avoiding a duplicate in the same project.

resource "google_secret_manager_secret" "github_app_private_key" {
  count = var.manage_github_app_secret ? 1 : 0

  project   = var.project_id
  secret_id = var.github_app_private_key_secret_id
  labels    = local.labels

  replication {
    auto {}
  }

  depends_on = [google_project_service.enabled]
}

resource "google_secret_manager_secret_version" "github_app_private_key" {
  count = var.manage_github_app_secret && local.github_app_private_key_present ? 1 : 0

  secret      = google_secret_manager_secret.github_app_private_key[0].id
  secret_data = var.github_app_private_key
}

resource "google_secret_manager_secret_iam_member" "cloud_build_github_app_secret_access" {
  count = var.manage_secret_accessor_bindings ? 1 : 0

  project   = var.project_id
  secret_id = var.manage_github_app_secret ? google_secret_manager_secret.github_app_private_key[0].secret_id : var.github_app_private_key_secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${local.cloud_build_service_account_email}"
}
