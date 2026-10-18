resource "google_service_account" "runtime" {
  count = var.manage_runtime_service_account ? 1 : 0

  project      = var.project_id
  account_id   = local.runtime_service_account_id
  display_name = "ArmadaCMS Cloud Run runtime"
  description  = "Runtime identity for the ArmadaCMS Cloud Run service."
}

resource "google_project_iam_member" "runtime_recaptcha_assessment" {
  count   = var.enable_recaptcha ? 1 : 0
  project = var.project_id
  role    = "roles/recaptchaenterprise.agent"
  member  = "serviceAccount:${var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)}"
}

resource "google_service_account" "deploy" {
  count = var.manage_deploy_service_account ? 1 : 0

  project      = var.project_id
  account_id   = local.deploy_service_account_id
  display_name = "ArmadaCMS production deployer"
  description  = "Least-privilege identity for ArmadaCMS production GitHub Actions releases."
}

# Deployment permissions are scoped to the repository, service and runtime identity.

resource "google_artifact_registry_repository_iam_member" "deploy_writer" {
  count = var.enable_actions_deployment ? 1 : 0

  project    = var.project_id
  location   = google_artifact_registry_repository.docker.location
  repository = google_artifact_registry_repository.docker.repository_id
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${local.deploy_service_account_email}"
}

resource "google_cloud_run_v2_service_iam_member" "deploy_developer" {
  count = var.enable_actions_deployment && var.deploy_cloud_run_service ? 1 : 0

  project  = var.project_id
  location = var.region
  name     = var.service_name
  role     = "roles/run.developer"
  member   = "serviceAccount:${local.deploy_service_account_email}"

  depends_on = [google_cloud_run_v2_service.armadacms]
}

resource "google_service_account_iam_member" "deploy_act_as_runtime" {
  count = var.enable_actions_deployment && var.manage_runtime_service_account ? 1 : 0

  service_account_id = google_service_account.runtime[0].name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${local.deploy_service_account_email}"
}
