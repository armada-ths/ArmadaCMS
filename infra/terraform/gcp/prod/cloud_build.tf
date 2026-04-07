resource "google_cloudbuild_trigger" "main_deploy" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project            = var.project_id
  location           = var.cloud_build_trigger_location
  name               = var.cloud_build_main_trigger_name
  description        = var.cloud_build_main_trigger_description
  filename           = var.cloud_build_config_filename
  service_account    = "projects/${var.project_id}/serviceAccounts/${local.cloud_build_service_account_email}"
  include_build_logs = var.cloud_build_include_logs

  github {
    owner = var.cloud_build_repo_owner
    name  = var.cloud_build_repo_name

    push {
      branch = var.cloud_build_main_branch_regex
    }
  }

  substitutions = {
    _TRIGGER_ID = var.cloud_build_main_trigger_id_substitution
  }

  depends_on = [google_project_service.enabled]
}

resource "google_cloudbuild_trigger" "pr_build" {
  count = var.manage_cloud_build_triggers ? 1 : 0

  project            = var.project_id
  location           = var.cloud_build_trigger_location
  name               = var.cloud_build_pr_trigger_name
  description        = var.cloud_build_pr_trigger_description
  filename           = var.cloud_build_config_filename
  service_account    = "projects/${var.project_id}/serviceAccounts/${local.cloud_build_service_account_email}"
  include_build_logs = var.cloud_build_include_logs

  github {
    owner = var.cloud_build_repo_owner
    name  = var.cloud_build_repo_name

    pull_request {
      branch = var.cloud_build_main_branch_regex
    }
  }

  substitutions = {
    _TRIGGER_ID = var.cloud_build_pr_trigger_id_substitution
  }

  depends_on = [google_project_service.enabled]
}
