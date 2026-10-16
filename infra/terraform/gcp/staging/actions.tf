resource "google_iam_workload_identity_pool" "github" {
  count                     = var.enable_actions_deployment ? 1 : 0
  project                   = var.project_id
  workload_identity_pool_id = "${var.name_prefix}-gha"
  display_name              = "CMS Actions Staging"
  depends_on                = [google_project_service.enabled]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  count                              = var.enable_actions_deployment ? 1 : 0
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github[0].workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  attribute_mapping = {
    "google.subject"          = "assertion.sub"
    "attribute.repository_id" = "assertion.repository_id"
  }
  attribute_condition = join(" && ", [
    "assertion.repository_owner_id == '6577346'",
    "assertion.repository_id == '961605060'",
    "assertion.ref == 'refs/heads/staging'",
    "assertion.event_name == 'push'",
    "assertion.sub == 'repo:armada-ths/ArmadaCMS:environment:Staging'",
    "assertion.workflow_ref == 'armada-ths/ArmadaCMS/.github/workflows/deploy.yml@refs/heads/staging'",
  ])
  oidc { issuer_uri = "https://token.actions.githubusercontent.com" }
}

resource "google_service_account_iam_member" "github_deploy" {
  count              = var.enable_actions_deployment ? 1 : 0
  service_account_id = "projects/${var.project_id}/serviceAccounts/${local.deploy_service_account_email}"
  role               = "roles/iam.workloadIdentityUser"
  member             = "principal://iam.googleapis.com/${google_iam_workload_identity_pool.github[0].name}/subject/repo:armada-ths/ArmadaCMS:environment:Staging"
}

output "github_environment_variables" {
  description = "Non-secret connection values for the Staging GitHub Environment."
  value = var.enable_actions_deployment ? {
    GCP_WORKLOAD_IDENTITY_PROVIDER = google_iam_workload_identity_pool_provider.github[0].name
    GCP_DEPLOY_SERVICE_ACCOUNT     = local.deploy_service_account_email
  } : {}
}

# Existing deploy identities and bindings retain their state addresses through this rename.
moved {
  from = google_service_account.cloud_build
  to   = google_service_account.deploy
}
moved {
  from = google_cloud_run_v2_service_iam_member.cloud_build_developer
  to   = google_cloud_run_v2_service_iam_member.deploy_developer
}
moved {
  from = google_service_account_iam_member.cloud_build_act_as_runtime
  to   = google_service_account_iam_member.deploy_act_as_runtime
}
moved {
  from = google_cloud_run_v2_job_iam_member.cloud_build_photo_worker_developer
  to   = google_cloud_run_v2_job_iam_member.deploy_photo_worker_developer
}

# Retired GitHub App secrets remain in Secret Manager; no secret values are read or destroyed.
removed {
  from = google_secret_manager_secret.github_app_private_key
  lifecycle { destroy = false }
}
removed {
  from = google_secret_manager_secret_version.github_app_private_key
  lifecycle { destroy = false }
}
