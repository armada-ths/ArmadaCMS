variable "project_id" {
  description = "Google Cloud project ID that hosts ArmadaCMS."
  type        = string
}

variable "region" {
  description = "Primary Google Cloud region for Cloud Run, Artifact Registry, and optional networking."
  type        = string
  default     = "europe-north2"
}

variable "environment" {
  description = "Environment label applied to provisioned resources."
  type        = string
  default     = "production"
}

variable "service_name" {
  description = "Cloud Run service name for ArmadaCMS."
  type        = string
  default     = "armadacms"
}

variable "name_prefix" {
  description = "Prefix used for related resources such as secrets, service accounts, and networking."
  type        = string
  default     = "armadacms"
}

variable "labels" {
  description = "Additional labels to merge into supported resources."
  type        = map(string)
  default     = {}
}

variable "artifact_registry_repository_id" {
  description = "Artifact Registry repository that stores ArmadaCMS container images."
  type        = string
  default     = "cloud-run-source-deploy"
}

variable "container_image_path" {
  description = "Path inside the Artifact Registry repository that Cloud Build pushes to."
  type        = string
  default     = "armadacms/armadacms"
}

variable "bootstrap_image" {
  description = "Optional fully-qualified image URI to use for the initial Cloud Run deployment. Leave empty to derive from Artifact Registry settings."
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
  description = "Optional GitHub App private key PEM content. Supplying this lets Terraform create the initial secret version, but it will also place the value in Terraform state."
  type        = string
  sensitive   = true
  default     = ""
}

variable "secret_values" {
  description = "Escape-hatch for seeding Secret Manager versions via Terraform. Intentionally left empty in practice — secret values are set directly in GCP Secret Manager (console or gcloud), never via Terraform, to keep them out of Terraform state."
  type        = map(string)
  sensitive   = true
  default     = {}
}

variable "revalidation_url" {
  description = "URL of the public site on-demand revalidation endpoint. Injected as a plain Cloud Run env var."
  type        = string
  default     = ""
}

variable "deploy_cloud_run_service" {
  description = "Whether Terraform should manage the Cloud Run service itself. Set this to false for the initial bootstrap if secrets have not been populated yet."
  type        = bool
  default     = false
}

variable "deletion_protection" {
  description = "Enable Cloud Run deletion protection. Recommended for production."
  type        = bool
  default     = true
}

variable "cloud_run_ingress" {
  description = "Ingress policy for the Cloud Run service."
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
  default     = "1"
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
  default     = 2
}

variable "cloud_build_service_account_email" {
  description = "Optional override for the Cloud Build service account email. Leave empty to use the project's default Cloud Build service account."
  type        = string
  default     = ""
}

variable "manage_cloud_build_triggers" {
  description = "Whether Terraform should manage the existing Cloud Build triggers for ArmadaCMS."
  type        = bool
  default     = false
}

variable "manage_secret_accessor_bindings" {
  description = "Whether Terraform should create per-secret Secret Manager accessor IAM bindings for the runtime and Cloud Build service accounts. Leave false to rely on broader project-level Secret Manager access if that already exists in the project."
  type        = bool
  default     = false
}

variable "manage_runtime_service_account" {
  description = "Whether Terraform should create and manage a dedicated runtime service account for Cloud Run."
  type        = bool
  default     = true
}

variable "cloud_run_service_account_email" {
  description = "Optional override for the Cloud Run runtime service account email. When manage_runtime_service_account is false and this is empty, Terraform falls back to the project's default compute service account."
  type        = string
  default     = ""
}

variable "enable_vpc_egress" {
  description = "Whether Cloud Run should use VPC egress and whether Terraform should manage the related network egress resources."
  type        = bool
  default     = false
}

variable "cloud_run_vpc_egress_mode" {
  description = "How Cloud Run should reach the VPC when enable_vpc_egress is true. Use DIRECT_VPC to match the current production setup, CONNECTOR for Serverless VPC Access, or NONE to disable VPC attachment."
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
  description = "Name of the VPC network used for Cloud Run egress and optional Cloud NAT resources."
  type        = string
  default     = "armadacms-serverless"
}

variable "vpc_subnetwork_name" {
  description = "Name of the subnetwork used for direct VPC egress and optional Cloud NAT resources."
  type        = string
  default     = "armadacms-serverless-europe-north2"
}

variable "manage_vpc_network_resources" {
  description = "Whether Terraform should create the VPC network and subnetwork resources instead of referencing existing ones."
  type        = bool
  default     = false
}

variable "enable_https_load_balancer" {
  description = "Whether Terraform should manage an external HTTPS load balancer in front of the Cloud Run service."
  type        = bool
  default     = false
}

variable "lb_reserve_global_ip" {
  description = "Whether to reserve and manage a global IP address for the HTTPS load balancer."
  type        = bool
  default     = true
}

variable "lb_enable_http_redirect" {
  description = "Whether to expose port 80 and redirect HTTP traffic to HTTPS."
  type        = bool
  default     = true
}

variable "lb_managed_certificate_domains" {
  description = "Domains to put on a Google-managed SSL certificate for the load balancer. Leave empty if you plan to reference existing certificate resources instead."
  type        = list(string)
  default     = []
}

variable "lb_existing_ssl_certificate_ids" {
  description = "Existing Compute SSL certificate self-links or IDs to attach to the HTTPS proxy instead of creating a managed certificate."
  type        = list(string)
  default     = []
}

# ── reCAPTCHA Enterprise ───────────────────────────────────────────────────────

variable "enable_recaptcha" {
  description = "Whether to create a reCAPTCHA Enterprise site key for the armada.nu website."
  type        = bool
  default     = false
}

variable "recaptcha_allowed_domains" {
  description = "Domains authorised to use the reCAPTCHA Enterprise site key."
  type        = list(string)
  default     = ["armada.nu"]
}

# ── Database overrides ────────────────────────────────────────────────────────
# When non-empty these override the values read from the Supabase workspace outputs.

variable "db_host" {
  description = "Override for DB_HOST. When empty, reads pooler_host from the armadacms-supabase-prod workspace output."
  type        = string
  default     = ""
}

variable "db_user" {
  description = "Override for DB_USER. When empty, reads pooler_user from the armadacms-supabase-prod workspace output."
  type        = string
  default     = ""
}

variable "db_name" {
  description = "Override for DB_NAME. When empty, reads db_name from the armadacms-supabase-prod workspace output."
  type        = string
  default     = ""
}
