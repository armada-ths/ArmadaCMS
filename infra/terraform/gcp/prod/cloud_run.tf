resource "google_cloud_run_v2_service" "armadacms" {
  count = var.deploy_cloud_run_service ? 1 : 0

  project             = var.project_id
  name                = var.service_name
  location            = var.region
  ingress             = var.cloud_run_ingress
  deletion_protection = var.deletion_protection
  labels              = local.labels

  template {
    service_account                  = var.manage_runtime_service_account ? google_service_account.runtime[0].email : (trimspace(var.cloud_run_service_account_email) != "" ? var.cloud_run_service_account_email : local.default_compute_service_account_email)
    timeout                          = "${var.cloud_run_timeout_seconds}s"
    max_instance_request_concurrency = var.cloud_run_concurrency
    execution_environment            = "EXECUTION_ENVIRONMENT_GEN2"

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    dynamic "vpc_access" {
      for_each = var.enable_vpc_egress && var.cloud_run_vpc_egress_mode != "NONE" ? [1] : []
      content {
        egress = var.vpc_egress

        connector = var.cloud_run_vpc_egress_mode == "CONNECTOR" ? google_vpc_access_connector.serverless[0].id : null

        dynamic "network_interfaces" {
          for_each = var.cloud_run_vpc_egress_mode == "DIRECT_VPC" ? [1] : []
          content {
            network    = local.selected_vpc_network_name
            subnetwork = local.selected_vpc_subnetwork_name
            tags       = var.cloud_run_vpc_network_tags
          }
        }
      }
    }

    containers {
      image = local.container_image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = var.cloud_run_cpu
          memory = var.cloud_run_memory
        }
        cpu_idle          = true
        startup_cpu_boost = var.enable_startup_cpu_boost
      }

      startup_probe {
        timeout_seconds   = 240
        period_seconds    = 240
        failure_threshold = 1

        tcp_socket {
          port = 8080
        }
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

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

  lifecycle {
    ignore_changes = [
      labels,
      template[0].labels,
      template[0].containers[0].image,
    ]
  }

  depends_on = [
    google_artifact_registry_repository.docker,
    google_secret_manager_secret_iam_member.runtime_secret_access,
    google_secret_manager_secret_version.app,
    google_compute_router_nat.serverless,
    google_vpc_access_connector.serverless,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  count = var.deploy_cloud_run_service && var.allow_unauthenticated ? 1 : 0

  project  = var.project_id
  location = google_cloud_run_v2_service.armadacms[0].location
  name     = google_cloud_run_v2_service.armadacms[0].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
