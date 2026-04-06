data "google_project" "current" {
  project_id = var.project_id
}

locals {
  base_labels = {
    app         = var.service_name
    environment = var.environment
    managed-by  = "terraform"
    repository  = "armadacms"
  }

  labels = merge(local.base_labels, var.labels)

  required_services = toset(concat(
    [
      "artifactregistry.googleapis.com",
      "cloudbuild.googleapis.com",
      "iam.googleapis.com",
      "run.googleapis.com",
      "secretmanager.googleapis.com",
      "serviceusage.googleapis.com",
    ],
    var.enable_vpc_egress ? [
      "compute.googleapis.com",
    ] : [],
    var.enable_vpc_egress && var.cloud_run_vpc_egress_mode == "CONNECTOR" ? [
      "vpcaccess.googleapis.com",
    ] : [],
    var.additional_project_services,
  ))

  secret_env_vars = {
    DB_PASSWORD           = "DB_PASSWORD"
    jwtsecret_laganda     = "jwtsecret_laganda"
    EVENTRO_API           = "EVENTRO_API"
    EVENTRO_FAIR_ID       = "EVENTRO_FAIR_ID"
    EVENTRO_ORG           = "EVENTRO_ORG"
    AWS_ACCESS_KEY_ID     = "AWS_ACCESS_KEY_ID"
    AWS_SECRET_ACCESS_KEY = "AWS_SECRET_ACCESS_KEY"
  }

  secret_value_keys = toset([
    for key in keys(nonsensitive(var.secret_values)) : key
    if contains(keys(local.secret_env_vars), key) && trimspace(nonsensitive(var.secret_values[key])) != ""
  ])

  github_app_private_key_present = trimspace(nonsensitive(var.github_app_private_key)) != ""

  plain_env_vars = merge(
    {
      DB_PORT                       = "5432"
      DB_USER                       = "postgres"
      DB_NAME                       = "armadacms"
      DB_SSLMODE                    = "require"
      S3_BUCKET                     = "armada-cms-files-e48105192c52"
      AWS_REGION                    = "eu-north-1"
      DB_MAX_OPEN_CONNS             = "10"
      DB_MAX_IDLE_CONNS             = "5"
      DB_CONN_MAX_LIFETIME_MINUTES  = "30"
      DB_CONN_MAX_IDLE_TIME_MINUTES = "10"
      AUDIT_LOG_RETENTION_DAYS      = "7"
    },
    var.additional_plain_env_vars,
  )

  cloud_build_service_account_email     = trimspace(var.cloud_build_service_account_email) != "" ? var.cloud_build_service_account_email : "${data.google_project.current.number}@cloudbuild.gserviceaccount.com"
  default_compute_service_account_email = "${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  runtime_service_account_id = substr(lower(replace("${var.name_prefix}-runtime", "_", "-")), 0, 30)

  artifact_registry_host = "${var.region}-docker.pkg.dev"
  container_image        = trimspace(var.bootstrap_image) != "" ? var.bootstrap_image : "${local.artifact_registry_host}/${var.project_id}/${var.artifact_registry_repository_id}/${var.container_image_path}:${var.bootstrap_image_tag}"
}
