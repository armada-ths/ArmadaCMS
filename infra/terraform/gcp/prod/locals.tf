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

  plain_env_vars = {
    DB_HOST                       = nonsensitive(data.tfe_outputs.aws_prod.values["rds_host"])
    DB_PORT                       = "5432"
    DB_USER                       = "postgres"
    DB_NAME                       = nonsensitive(data.tfe_outputs.aws_prod.values["rds_db_name"])
    DB_SSLMODE                    = "require"
    S3_BUCKET                     = nonsensitive(data.tfe_outputs.aws_prod.values["s3_bucket_name"])
    AWS_REGION                    = nonsensitive(data.tfe_outputs.aws_prod.values["s3_bucket_region"])
    DB_MAX_OPEN_CONNS             = "10"
    DB_MAX_IDLE_CONNS             = "5"
    DB_CONN_MAX_LIFETIME_MINUTES  = "30"
    DB_CONN_MAX_IDLE_TIME_MINUTES = "10"
    AUDIT_LOG_RETENTION_DAYS      = "7"
  }

  cloud_build_service_account_email     = trimspace(var.cloud_build_service_account_email) != "" ? var.cloud_build_service_account_email : "${data.google_project.current.number}@cloudbuild.gserviceaccount.com"
  default_compute_service_account_email = "${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  runtime_service_account_id = substr(lower(replace("${var.name_prefix}-runtime", "_", "-")), 0, 30)

  artifact_registry_host = "${var.region}-docker.pkg.dev"
  container_image        = trimspace(var.bootstrap_image) != "" ? var.bootstrap_image : "${local.artifact_registry_host}/${var.project_id}/${var.artifact_registry_repository_id}/${var.container_image_path}:${var.bootstrap_image_tag}"

  # ── Derived resource names ──────────────────────────────────────────────────────
  # All names are derived from var.name_prefix so a single rename propagates
  # everywhere. Matches the resources already deployed in production.

  nat_router_name    = "${var.name_prefix}-router"
  nat_name           = "${var.name_prefix}-nat"
  nat_ip_name        = "${var.name_prefix}-nat-ip"
  vpc_subnet_cidr    = "10.8.0.0/24"
  vpc_connector_name = "${var.name_prefix}-serverless"
  vpc_connector_cidr = "10.8.0.0/28"
  vpc_egress         = "ALL_TRAFFIC"

  lb_neg_name                   = "${var.name_prefix}-neg"
  lb_backend_name               = "${var.name_prefix}-backend"
  lb_url_map_name               = "${var.name_prefix}-lb"
  lb_redirect_url_map_name      = "${var.name_prefix}-https-redirect"
  lb_target_https_proxy_name    = "${var.name_prefix}-lb-target-proxy"
  lb_target_http_proxy_name     = "${var.name_prefix}-https-target-proxy"
  lb_https_forwarding_rule_name = "${var.name_prefix}-https"
  lb_http_forwarding_rule_name  = "${var.name_prefix}-https-forwarding-rule"
  lb_global_address_name        = "${var.name_prefix}-lb-ip"
  lb_certificate_name           = "cms-armada-nu-cert"

  cloud_build_main_trigger_name        = "${var.name_prefix}-main-deploy"
  cloud_build_main_trigger_description = "Build and deploy to Cloud Run service ${var.service_name} on push to \"^main$\""
  cloud_build_main_trigger_id          = "${var.name_prefix}-main"
  cloud_build_pr_trigger_name          = "${var.name_prefix}-pr-build"
  cloud_build_pr_trigger_description   = "Build and push ${var.service_name} PR image tagged pr-<PR number>"
  cloud_build_pr_trigger_id            = "${var.name_prefix}-pr"
}
