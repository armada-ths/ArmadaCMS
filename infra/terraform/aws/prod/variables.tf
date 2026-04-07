variable "region" {
  description = "AWS region for all resources."
  type        = string
  default     = "eu-north-1"
}

variable "environment" {
  description = "Environment label applied to resource tags."
  type        = string
  default     = "production"
}

variable "cloud_run_nat_ip_override" {
  description = "Override for the Cloud Run NAT outbound IP. Leave null (default) to read it automatically from the armadacms-gcp-prod HCP Terraform workspace output 'static_egress_ip'."
  type        = string
  default     = null
}

variable "db_password" {
  description = "Master password for the RDS PostgreSQL instance. Sensitive — pass via TF_VAR_db_password or an HCP Terraform variable. Terraform records this in state; use an encrypted remote backend."
  type        = string
  sensitive   = true
}
