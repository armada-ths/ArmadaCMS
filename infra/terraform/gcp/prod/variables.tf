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
  description = "Optional map of ArmadaCMS runtime secret values keyed by environment variable name. Supplying values lets Terraform create secret versions, but the values will be stored in Terraform state."
  type        = map(string)
  sensitive   = true
  default     = {}
}

variable "additional_plain_env_vars" {
  description = "Extra non-secret environment variables to inject into the Cloud Run container."
  type        = map(string)
  default     = {}
}

variable "deploy_cloud_run_service" {
  description = "Whether Terraform should manage the Cloud Run service itself. Set this to false for the initial bootstrap if secrets have not been populated yet."
  type        = bool
  default     = false
}

variable "allow_unauthenticated" {
  description = "Whether to allow public unauthenticated invocation of the Cloud Run service."
  type        = bool
  default     = true
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

variable "enable_startup_cpu_boost" {
  description = "Whether to enable Cloud Run startup CPU boost for faster cold starts."
  type        = bool
  default     = true
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

variable "cloud_build_trigger_location" {
  description = "Location of the Cloud Build triggers. Existing ArmadaCMS triggers live in the global location."
  type        = string
  default     = "global"
}

variable "cloud_build_config_filename" {
  description = "Path to the Cloud Build configuration file used by the triggers."
  type        = string
  default     = "cloudbuild.yaml"
}

variable "cloud_build_repo_owner" {
  description = "GitHub owner for the ArmadaCMS repository watched by Cloud Build triggers."
  type        = string
  default     = "armada-ths"
}

variable "cloud_build_repo_name" {
  description = "GitHub repository name watched by Cloud Build triggers."
  type        = string
  default     = "ArmadaCMS"
}

variable "cloud_build_main_branch_regex" {
  description = "Regex for the main-branch deploy trigger."
  type        = string
  default     = "^main$"
}

variable "cloud_build_main_trigger_name" {
  description = "Name of the Cloud Build trigger that deploys main to production."
  type        = string
  default     = "armadacms-main-deploy"
}

variable "cloud_build_main_trigger_description" {
  description = "Description of the production Cloud Build trigger."
  type        = string
  default     = "Build and deploy to Cloud Run service armadacms on push to \"^main$\""
}

variable "cloud_build_main_trigger_id_substitution" {
  description = "_TRIGGER_ID substitution value used by the production deploy trigger."
  type        = string
  default     = "armadacms-main"
}

variable "cloud_build_pr_trigger_name" {
  description = "Name of the Cloud Build trigger that builds PR images."
  type        = string
  default     = "armadacms-pr-build"
}

variable "cloud_build_pr_trigger_description" {
  description = "Description of the PR Cloud Build trigger."
  type        = string
  default     = "Build and push ArmadaCMS PR image tagged pr-<PR number>"
}

variable "cloud_build_pr_trigger_id_substitution" {
  description = "_TRIGGER_ID substitution value used by the PR build trigger."
  type        = string
  default     = "armadacms-pr"
}

variable "cloud_build_include_logs" {
  description = "How Cloud Build should include build logs in trigger status reporting."
  type        = string
  default     = "INCLUDE_BUILD_LOGS_WITH_STATUS"
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

variable "vpc_subnet_cidr" {
  description = "CIDR range for the managed subnetwork used by Cloud NAT when manage_vpc_network_resources is true."
  type        = string
  default     = "10.20.0.0/24"
}

variable "vpc_connector_name" {
  description = "Name of the serverless VPC access connector."
  type        = string
  default     = "armadacms-serverless"
}

variable "vpc_connector_cidr" {
  description = "CIDR range reserved for the serverless VPC access connector. Must not overlap the subnetwork CIDR."
  type        = string
  default     = "10.8.0.0/28"
}

variable "vpc_egress" {
  description = "Traffic egress setting when the Cloud Run service is attached to the VPC."
  type        = string
  default     = "ALL_TRAFFIC"

  validation {
    condition = contains([
      "ALL_TRAFFIC",
      "PRIVATE_RANGES_ONLY",
    ], var.vpc_egress)
    error_message = "vpc_egress must be ALL_TRAFFIC or PRIVATE_RANGES_ONLY."
  }
}

variable "nat_router_name" {
  description = "Name of the Cloud Router used for Cloud NAT."
  type        = string
  default     = "armadacms-serverless-nat-router"
}

variable "nat_name" {
  description = "Name of the Cloud NAT configuration used for stable egress."
  type        = string
  default     = "armadacms-nat"
}

variable "nat_ip_name" {
  description = "Name of the reserved external IP address used for Cloud NAT egress."
  type        = string
  default     = "armadacms-egress-ip"
}

variable "cloud_run_vpc_network_tags" {
  description = "Optional network tags applied to Cloud Run's direct VPC interface when cloud_run_vpc_egress_mode is DIRECT_VPC."
  type        = list(string)
  default     = []
}

variable "additional_project_services" {
  description = "Extra Google APIs to enable in the target project."
  type        = list(string)
  default     = []
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

variable "lb_serverless_neg_name" {
  description = "Name of the serverless NEG that points at the Cloud Run service."
  type        = string
  default     = "armadacms-neg"
}

variable "lb_backend_service_name" {
  description = "Name of the global backend service used by the HTTPS load balancer."
  type        = string
  default     = "armadacms-backend"
}

variable "lb_url_map_name" {
  description = "Name of the primary URL map for the HTTPS load balancer."
  type        = string
  default     = "armadacms-lb"
}

variable "lb_redirect_url_map_name" {
  description = "Name of the URL map used for HTTP to HTTPS redirects."
  type        = string
  default     = "armadacms-https-redirect"
}

variable "lb_target_https_proxy_name" {
  description = "Name of the target HTTPS proxy."
  type        = string
  default     = "armadacms-lb-target-proxy"
}

variable "lb_target_http_proxy_name" {
  description = "Name of the target HTTP proxy used for redirects."
  type        = string
  default     = "armadacms-https-target-proxy"
}

variable "lb_https_forwarding_rule_name" {
  description = "Name of the global forwarding rule for HTTPS traffic."
  type        = string
  default     = "armadacms-https"
}

variable "lb_http_forwarding_rule_name" {
  description = "Name of the global forwarding rule for HTTP traffic."
  type        = string
  default     = "armadacms-https-forwarding-rule"
}

variable "lb_global_address_name" {
  description = "Name of the global IP address resource for the load balancer."
  type        = string
  default     = "armadacms-lb-ip"
}

variable "lb_managed_certificate_name" {
  description = "Name of the Google-managed SSL certificate resource when Terraform manages the certificate."
  type        = string
  default     = "cms-armada-nu-cert"
}

variable "lb_backend_timeout_seconds" {
  description = "Timeout for the global backend service used by the external HTTPS load balancer."
  type        = number
  default     = 30
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
