resource "google_cloudbuild_trigger" "main_deploy" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project            = var.project_id
  location           = "global"
  name               = local.cloud_build_main_trigger_name
  description        = local.cloud_build_main_trigger_description
  filename           = "cloudbuild.yaml"
  service_account    = "projects/${var.project_id}/serviceAccounts/${local.cloud_build_service_account_email}"
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"

  github {
    owner = "armada-ths"
    name  = "ArmadaCMS"

    push {
      branch = "^main$"
    }
  }

  substitutions = {
    _TRIGGER_ID = local.cloud_build_main_trigger_id
  }

  depends_on = [google_project_service.enabled]
}

resource "google_cloudbuild_trigger" "pr_build" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project            = var.project_id
  location           = "global"
  name               = local.cloud_build_pr_trigger_name
  description        = local.cloud_build_pr_trigger_description
  filename           = "cloudbuild.yaml"
  service_account    = "projects/${var.project_id}/serviceAccounts/${local.cloud_build_service_account_email}"
  include_build_logs = "INCLUDE_BUILD_LOGS_WITH_STATUS"

  github {
    owner = "armada-ths"
    name  = "ArmadaCMS"

    pull_request {
      branch = "^main$"
    }
  }

  substitutions = {
    _TRIGGER_ID = local.cloud_build_pr_trigger_id
  }

  depends_on = [google_project_service.enabled]
}
