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
    var.enable_recaptcha ? [
      "apikeys.googleapis.com",
      "recaptchaenterprise.googleapis.com",
    ] : [],
    var.enable_vpc_egress ? [
      "compute.googleapis.com",
    ] : [],
    var.enable_vpc_egress && var.cloud_run_vpc_egress_mode == "CONNECTOR" ? [
      "vpcaccess.googleapis.com",
    ] : [],
  ))

  # Secret IDs are prefixed with name_prefix to avoid colliding with production
  # secrets in the same GCP project.
  secret_env_vars = merge(
    {
      DB_PASSWORD            = "${var.name_prefix}-DB_PASSWORD"
      jwtsecret_laganda      = "${var.name_prefix}-jwtsecret_laganda"
      EVENTRO_API            = "${var.name_prefix}-EVENTRO_API"
      EVENTRO_FAIR_ID        = "${var.name_prefix}-EVENTRO_FAIR_ID"
      EVENTRO_ORG            = "${var.name_prefix}-EVENTRO_ORG"
      INITIAL_ADMIN_PASSWORD = "${var.name_prefix}-INITIAL_ADMIN_PASSWORD"
    },
    var.storage_provider == "s3" ? {
      AWS_ACCESS_KEY_ID     = "${var.name_prefix}-AWS_ACCESS_KEY_ID"
      AWS_SECRET_ACCESS_KEY = "${var.name_prefix}-AWS_SECRET_ACCESS_KEY"
    } : {},
    var.storage_provider == "supabase" ? {
      SUPABASE_STORAGE_ACCESS_KEY_ID     = "${var.name_prefix}-SUPABASE_STORAGE_ACCESS_KEY_ID"
      SUPABASE_STORAGE_SECRET_ACCESS_KEY = "${var.name_prefix}-SUPABASE_STORAGE_SECRET_ACCESS_KEY"
    } : {},
  )

  secret_value_keys = toset([
    for key in keys(nonsensitive(var.secret_values)) : key
    if contains(keys(local.secret_env_vars), key) && trimspace(nonsensitive(var.secret_values[key])) != ""
  ])

  github_app_private_key_present = trimspace(nonsensitive(var.github_app_private_key)) != ""

  # DB values come directly from Supabase variables. Storage values are selected
  # by storage_provider so staging can be switched from AWS S3 to Supabase
  # Storage without changing application code.
  plain_env_vars = merge(
    {
      DB_HOST                       = var.db_host
      DB_PORT                       = "5432"
      DB_USER                       = var.db_user
      DB_NAME                       = var.db_name
      DB_SSLMODE                    = "require"
      STORAGE_PROVIDER              = var.storage_provider
      DB_MAX_OPEN_CONNS             = "5"
      DB_MAX_IDLE_CONNS             = "2"
      DB_CONN_MAX_LIFETIME_MINUTES  = "30"
      DB_CONN_MAX_IDLE_TIME_MINUTES = "10"
      AUDIT_LOG_RETENTION_DAYS      = "7"
      INITIAL_ADMIN_USERNAME        = var.initial_admin_username
    },
    var.storage_provider == "s3" ? {
      S3_BUCKET  = nonsensitive(data.tfe_outputs.aws_staging[0].values["s3_bucket_name"])
      AWS_REGION = nonsensitive(data.tfe_outputs.aws_staging[0].values["s3_bucket_region"])
    } : {},
    var.storage_provider == "supabase" ? {
      SUPABASE_URL                 = var.supabase_url
      SUPABASE_STORAGE_S3_ENDPOINT = var.supabase_storage_s3_endpoint
      SUPABASE_STORAGE_BUCKET      = var.supabase_storage_bucket
      SUPABASE_STORAGE_REGION      = var.supabase_storage_region
    } : {},
  )

  cloud_build_service_account_email     = trimspace(var.cloud_build_service_account_email) != "" ? var.cloud_build_service_account_email : "${data.google_project.current.number}@cloudbuild.gserviceaccount.com"
  default_compute_service_account_email = "${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  runtime_service_account_id = substr(lower(replace("${var.name_prefix}-runtime", "_", "-")), 0, 30)

  artifact_registry_host = "${var.region}-docker.pkg.dev"
  container_image        = trimspace(var.bootstrap_image) != "" ? var.bootstrap_image : "${var.prod_artifact_registry_host}/${var.project_id}/${var.prod_artifact_registry_repository_id}/${var.container_image_path}:${var.bootstrap_image_tag}"

  # ── Derived resource names ──────────────────────────────────────────────────
  nat_router_name    = "${var.name_prefix}-router"
  nat_name           = "${var.name_prefix}-nat"
  nat_ip_name        = "${var.name_prefix}-nat-ip"
  vpc_subnet_cidr    = "10.9.0.0/24"
  vpc_connector_name = "${var.name_prefix}-serverless"
  vpc_connector_cidr = "10.9.0.0/28"
  vpc_egress         = "ALL_TRAFFIC"

  # Load balancer names (not used for staging but kept for parity)
  lb_neg_name                   = "${var.name_prefix}-neg"
  lb_backend_name               = "${var.name_prefix}-backend"
  lb_url_map_name               = "${var.name_prefix}-lb"
  lb_redirect_url_map_name      = "${var.name_prefix}-https-redirect"
  lb_target_https_proxy_name    = "${var.name_prefix}-lb-target-proxy"
  lb_target_http_proxy_name     = "${var.name_prefix}-https-target-proxy"
  lb_https_forwarding_rule_name = "${var.name_prefix}-https"
  lb_http_forwarding_rule_name  = "${var.name_prefix}-https-forwarding-rule"
  lb_global_address_name        = "${var.name_prefix}-lb-ip"
  lb_certificate_name           = "${var.name_prefix}-cert"

  cloud_build_staging_trigger_name           = "${var.name_prefix}-staging-deploy"
  cloud_build_staging_trigger_description    = "Build and deploy to Cloud Run service ${var.service_name} on push to \"^staging$\""
  cloud_build_staging_trigger_id             = "${var.name_prefix}-staging"
  cloud_build_staging_pr_trigger_name        = "${var.name_prefix}-staging-pr-build"
  cloud_build_staging_pr_trigger_description = "Build and push ${var.service_name} PR image tagged pr-<PR number> for PRs targeting \"^staging$\""
  cloud_build_staging_pr_trigger_id          = "${var.name_prefix}-staging-pr"
}
