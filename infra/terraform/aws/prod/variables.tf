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

variable "cloud_run_nat_ip" {
  description = "Static outbound IP of the Cloud Run NAT gateway. Used to allowlist PostgreSQL (port 5432) on the RDS security group and restrict S3 PutObject access via IAM policy."
  type        = string
  default     = "34.51.249.94"
}

variable "db_password" {
  description = "Master password for the RDS PostgreSQL instance. Sensitive — pass via TF_VAR_db_password or an HCP Terraform variable. Terraform records this in state; use an encrypted remote backend."
  type        = string
  sensitive   = true
}
