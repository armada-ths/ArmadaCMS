# Staging branch deploy trigger — tracks "staging" branch only.
# No PR trigger is needed for the staging environment.
#
# The substitutions block overrides cloudbuild.yaml defaults so the build
# targets the staging Artifact Registry repo, Cloud Run service, and region.

resource "google_cloudbuild_trigger" "staging_deploy" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project            = var.project_id
  location           = "global"
  name               = local.cloud_build_staging_trigger_name
  description        = local.cloud_build_staging_trigger_description
  filename           = "cloudbuild.yaml"
  service_account    = "projects/${var.project_id}/serviceAccounts/${local.cloud_build_service_account_email}"
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"

  github {
    owner = "armada-ths"
    name  = "ArmadaCMS"

    push {
      branch = "^staging$"
    }
  }

  substitutions = {
    _TRIGGER_ID    = local.cloud_build_staging_trigger_id
    _DEPLOY_REGION = var.region
    # Staging images are pushed to the Artifact Registry repository managed by
    # the production Terraform root; staging does not own a separate repository.
    _AR_HOSTNAME        = var.prod_artifact_registry_host
    _AR_REPOSITORY      = var.prod_artifact_registry_repository_id
    _AR_PROJECT_ID      = var.project_id
    _SERVICE_NAME       = var.service_name
    _GITHUB_ENVIRONMENT = "Staging"
  }

  depends_on = [google_project_service.enabled]
}

# Untrusted PR validation trigger — fires on PRs targeting staging. It uses a
# secret-free build config and an identity that cannot publish or deploy.

resource "google_cloudbuild_trigger" "staging_pr_build" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project            = var.project_id
  location           = "global"
  name               = local.cloud_build_staging_pr_trigger_name
  description        = local.cloud_build_staging_pr_trigger_description
  filename           = "cloudbuild-pr.yaml"
  service_account    = "projects/${var.project_id}/serviceAccounts/${local.cloud_build_pr_service_account_email}"
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"

  github {
    owner = "armada-ths"
    name  = "ArmadaCMS"

    pull_request {
      branch          = "^staging$"
      comment_control = "COMMENTS_ENABLED_FOR_EXTERNAL_CONTRIBUTORS_ONLY"
    }
  }

  substitutions = {
    _SERVICE_NAME = var.service_name
  }

  depends_on = [google_project_service.enabled]
}
