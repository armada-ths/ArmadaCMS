resource "google_service_account" "runtime" {
  count = var.manage_runtime_service_account ? 1 : 0

  project      = var.project_id
  account_id   = local.runtime_service_account_id
  display_name = "ArmadaCMS Cloud Run runtime"
  description  = "Runtime identity for the ArmadaCMS Cloud Run service."
}

resource "google_service_account" "cloud_build" {
  count = var.manage_cloud_build_service_account ? 1 : 0

  project      = var.project_id
  account_id   = local.cloud_build_service_account_id
  display_name = "ArmadaCMS production deployer"
  description  = "Least-privilege identity for ArmadaCMS production Cloud Build triggers."
}

# Pull request builds execute contributor-controlled code. Keep this identity
# separate from the deployer so PRs cannot read secrets, deploy Cloud Run, or
# impersonate the runtime service account.
resource "google_service_account" "cloud_build_pr" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project      = var.project_id
  account_id   = local.cloud_build_pr_service_account_id
  display_name = "ArmadaCMS pull request builder"
  description  = "Unprivileged builder for contributor pull requests."
}

# Cloud Logging is project-scoped. Artifact Registry and Cloud Run permissions
# are granted directly on the repository and service below.
resource "google_project_iam_member" "cloud_build_log_writer" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${local.cloud_build_service_account_email}"

  depends_on = [google_project_service.enabled]
}

resource "google_project_iam_member" "cloud_build_pr_log_writer" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${local.cloud_build_pr_service_account_email}"

  depends_on = [google_project_service.enabled]
}

resource "google_artifact_registry_repository_iam_member" "cloud_build_writer" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project    = var.project_id
  location   = google_artifact_registry_repository.docker.location
  repository = google_artifact_registry_repository.docker.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${local.cloud_build_service_account_email}"
}

resource "google_cloud_run_v2_service_iam_member" "cloud_build_developer" {
  count = var.manage_cloud_build_triggers && var.deploy_cloud_run_service ? 1 : 0

  project  = var.project_id
  location = var.region
  name     = var.service_name
  role     = "roles/run.developer"
  member   = "serviceAccount:${local.cloud_build_service_account_email}"

  depends_on = [google_cloud_run_v2_service.armadacms]
}

resource "google_service_account_iam_member" "cloud_build_act_as_runtime" {
  count = var.manage_cloud_build_triggers && var.manage_runtime_service_account ? 1 : 0

  service_account_id = google_service_account.runtime[0].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${local.cloud_build_service_account_email}"
}
