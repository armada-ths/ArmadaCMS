variable "project_id" {
  description = "Google Cloud project ID that hosts ArmadaCMS."
  type        = string
}

variable "region" {
  description = "Primary Google Cloud region for Cloud Run, Artifact Registry, and optional networking. Must support Cloud Run custom domain mappings (europe-north1 recommended)."
  type        = string
  default     = "europe-north1"
}

variable "environment" {
  description = "Environment label applied to provisioned resources."
  type        = string
  default     = "staging"
}

variable "service_name" {
  description = "Cloud Run service name for ArmadaCMS staging."
  type        = string
  default     = "armadacms-staging"
}

variable "name_prefix" {
  description = "Prefix used for related resources such as secrets, service accounts, and networking."
  type        = string
  default     = "armadacms-staging"
}

variable "labels" {
  description = "Additional labels to merge into supported resources."
  type        = map(string)
  default     = {}
}

variable "prod_artifact_registry_host" {
  description = "Hostname of the production Artifact Registry repository that Cloud Build writes images to. Staging and prod share this repo so a commit built on staging is reused by the prod trigger."
  type        = string
  default     = "europe-north2-docker.pkg.dev"
}

variable "prod_artifact_registry_repository_id" {
  description = "Repository ID of the shared production Artifact Registry that both staging and prod Cloud Build triggers push images to."
  type        = string
  default     = "cloud-run-source-deploy"
}

variable "container_image_path" {
  description = "Path inside the Artifact Registry repository that Cloud Build pushes to."
  type        = string
  default     = "armadacms/armadacms"
}

variable "bootstrap_image" {
  description = "Optional fully-qualified image URI for the initial Cloud Run deployment. Leave empty to derive from Artifact Registry settings. Set this to an existing image (e.g. gcr.io/cloudrun/hello) before the first Cloud Build run creates a staging image."
  type        = string
  default     = ""
}

variable "bootstrap_image_tag" {
  description = "Image tag used when bootstrap_image is not set explicitly."
  type        = string
  default     = "bootstrap"
}

variable "github_app_private_key_secret_id" {
  description = "Secret Manager secret ID used by cloudbuild.yaml for GitHub deployment tracking."
  type        = string
  default     = "github-app-private-key"
}

variable "github_app_private_key" {
  description = "Optional GitHub App private key PEM content. Leave empty to reuse the existing secret created by the production workspace."
  type        = string
  sensitive   = true
  default     = ""
}

variable "manage_github_app_secret" {
  description = "Whether Terraform should create the github-app-private-key Secret Manager secret. Set to false to reuse the secret already created by the production workspace."
  type        = bool
  default     = false
}

variable "secret_values" {
  description = "Optional map of ArmadaCMS runtime secret values keyed by environment variable name. Supplying values lets Terraform create secret versions, but the values will be stored in Terraform state."
  type        = map(string)
  sensitive   = true
  default     = {}
}

variable "deploy_cloud_run_service" {
  description = "Whether Terraform should manage the Cloud Run service itself. Set to false until both the staging Cloud Build has produced a first image and the AWS staging workspace outputs are available."
  type        = bool
  default     = false
}

variable "deletion_protection" {
  description = "Enable Cloud Run deletion protection."
  type        = bool
  default     = false
}

variable "cloud_run_ingress" {
  description = "Ingress policy for the Cloud Run service. Must be INGRESS_TRAFFIC_ALL when using Cloud Run custom domain mappings."
  type        = string
  default     = "INGRESS_TRAFFIC_ALL"

  validation {
    condition = contains([
      "INGRESS_TRAFFIC_ALL",
      "INGRESS_TRAFFIC_INTERNAL_ONLY",
      "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER",
    ], var.cloud_run_ingress)
    error_message = "cloud_run_ingress must be a valid Cloud Run ingress enum value."
  }
}

variable "cloud_run_cpu" {
  description = "CPU limit for the ArmadaCMS Cloud Run container."
  type        = string
  default     = "1000m"
}

variable "cloud_run_memory" {
  description = "Memory limit for the ArmadaCMS Cloud Run container."
  type        = string
  default     = "512Mi"
}

variable "cloud_run_timeout_seconds" {
  description = "Request timeout for the ArmadaCMS Cloud Run service in seconds."
  type        = number
  default     = 120
}

variable "cloud_run_concurrency" {
  description = "Maximum concurrent requests per Cloud Run instance."
  type        = number
  default     = 10
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances."
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances."
  type        = number
  default     = 1
}

variable "cloud_build_service_account_email" {
  description = "Optional override for the Cloud Build service account email. Leave empty to use the project's default Cloud Build service account."
  type        = string
  default     = ""
}

variable "manage_cloud_build_triggers" {
  description = "Whether Terraform should manage the Cloud Build triggers for ArmadaCMS staging."
  type        = bool
  default     = false
}

variable "manage_secret_accessor_bindings" {
  description = "Whether Terraform should create per-secret Secret Manager accessor IAM bindings. Leave false to rely on broader project-level access."
  type        = bool
  default     = false
}

variable "manage_runtime_service_account" {
  description = "Whether Terraform should create and manage a dedicated runtime service account for Cloud Run."
  type        = bool
  default     = false
}

variable "cloud_run_service_account_email" {
  description = "Optional override for the Cloud Run runtime service account email."
  type        = string
  default     = ""
}

variable "enable_vpc_egress" {
  description = "Whether Cloud Run should use VPC egress for a stable outbound IP (required for S3 IP restriction)."
  type        = bool
  default     = true
}

variable "cloud_run_vpc_egress_mode" {
  description = "How Cloud Run should reach the VPC when enable_vpc_egress is true."
  type        = string
  default     = "DIRECT_VPC"

  validation {
    condition = contains([
      "NONE",
      "DIRECT_VPC",
      "CONNECTOR",
    ], var.cloud_run_vpc_egress_mode)
    error_message = "cloud_run_vpc_egress_mode must be NONE, DIRECT_VPC, or CONNECTOR."
  }
}

variable "vpc_network_name" {
  description = "Name of the VPC network used for Cloud Run egress."
  type        = string
  default     = "default"
}

variable "vpc_subnetwork_name" {
  description = "Name of the subnetwork used for direct VPC egress and Cloud NAT."
  type        = string
  default     = "default"
}

variable "manage_vpc_network_resources" {
  description = "Whether Terraform should create the VPC network and subnetwork instead of referencing existing ones."
  type        = bool
  default     = false
}

# ── Custom domain mapping ──────────────────────────────────────────────────────

variable "enable_domain_mapping" {
  description = "Whether to create a Cloud Run custom domain mapping. Requires cloud_run_ingress = INGRESS_TRAFFIC_ALL and a supported region."
  type        = bool
  default     = false
}

variable "domain_mapping_hostname" {
  description = "Hostname for the Cloud Run custom domain mapping (e.g. staging.cms.armada.nu)."
  type        = string
  default     = ""
}

# ── Supabase database ─────────────────────────────────────────────────────────

variable "db_host" {
  description = "PostgreSQL database hostname (Supabase host for staging)."
  type        = string
  default     = "db.yfybmnqzclpmpncyfmdc.supabase.co"
}

variable "db_user" {
  description = "PostgreSQL database user."
  type        = string
  default     = "postgres"
}

variable "db_name" {
  description = "PostgreSQL database name."
  type        = string
  default     = "postgres"
}

variable "initial_admin_username" {
  description = "Username for the initial admin user seeded on first startup when the DB has no users. Leave empty to skip seeding."
  type        = string
  default     = ""
}

# ── Load balancer (disabled for staging — kept for future parity) ─────────────

variable "enable_https_load_balancer" {
  description = "Whether Terraform should manage an external HTTPS load balancer. Disabled for staging in favour of Cloud Run domain mappings."
  type        = bool
  default     = false
}

variable "lb_reserve_global_ip" {
  type    = bool
  default = true
}

variable "lb_enable_http_redirect" {
  type    = bool
  default = true
}

variable "lb_managed_certificate_domains" {
  type    = list(string)
  default = []
}

variable "lb_existing_ssl_certificate_ids" {
  type    = list(string)
  default = []
}
