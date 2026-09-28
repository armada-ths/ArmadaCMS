resource "google_cloud_run_v2_job" "photo_worker" {
  count               = var.deploy_cloud_run_service ? 1 : 0
  project             = var.project_id
  location            = var.region
  name                = "${var.service_name}-photo-worker"
  deletion_protection = var.deletion_protection
  labels              = local.labels

  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account       = var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)
      timeout               = "3600s"
      max_retries           = 1
      execution_environment = "EXECUTION_ENVIRONMENT_GEN2"
      dynamic "vpc_access" {
        for_each = var.enable_vpc_egress && var.cloud_run_vpc_egress_mode != "NONE" ? [1] : []
        content {
          egress    = local.vpc_egress
          connector = var.cloud_run_vpc_egress_mode == "CONNECTOR" ? google_vpc_access_connector.serverless[0].id : null
          dynamic "network_interfaces" {
            for_each = var.cloud_run_vpc_egress_mode == "DIRECT_VPC" ? [1] : []
            content {
              network    = local.selected_vpc_network_name
              subnetwork = local.selected_vpc_subnetwork_name
              tags       = []
            }
          }
        }
      }
      containers {
        image = local.container_image
        resources { limits = { cpu = "1", memory = "1Gi" } }
        env {
          name  = "PHOTO_WORKER_MODE"
          value = "1"
        }
        dynamic "env" {
          for_each = local.plain_env_vars
          content {
            name  = env.key
            value = env.value
          }
        }
        dynamic "env" {
          for_each = local.secret_env_vars
          content {
            name = env.key
            value_source {
              secret_key_ref {
                secret  = google_secret_manager_secret.app[env.key].secret_id
                version = "latest"
              }
            }
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [template[0].template[0].containers[0].image]
  }

  depends_on = [google_secret_manager_secret_iam_member.runtime_secret_access, google_secret_manager_secret_version.app]
}

resource "google_cloud_run_v2_job_iam_member" "cloud_build_photo_worker_developer" {
  count    = var.manage_cloud_build_triggers && var.deploy_cloud_run_service ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_job.photo_worker[0].name
  role     = "roles/run.developer"
  member   = "serviceAccount:${local.cloud_build_service_account_email}"
}

resource "google_cloud_run_v2_job_iam_member" "photo_worker_invoker" {
  count    = var.deploy_cloud_run_service ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_job.photo_worker[0].name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)}"
}

resource "google_cloud_scheduler_job" "photo_worker_retry" {
  count       = var.deploy_cloud_run_service ? 1 : 0
  project     = var.project_id
  region      = var.region
  name        = "${var.service_name}-photo-worker-retry"
  description = "Retry queued photo exports and enforce retention"
  schedule    = "*/15 * * * *"
  time_zone   = "Europe/Stockholm"
  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/${google_cloud_run_v2_job.photo_worker[0].id}:run"
    oauth_token {
      service_account_email = var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)
    }
  }
  depends_on = [google_cloud_run_v2_job_iam_member.photo_worker_invoker, google_project_service.enabled]
}
