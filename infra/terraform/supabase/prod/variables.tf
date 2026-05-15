variable "organization_id" {
  description = "Supabase organization slug that owns the ArmadaCMS production project. Despite the name, the provider expects the organization slug string shown in the Supabase dashboard."
  type        = string
}

variable "project_ref" {
  description = "Supabase production project reference to import and manage."
  type        = string
  default     = "rsdjnixgxqauonaofrwr"
}

variable "project_name" {
  description = "Human-readable name of the Supabase production project. Must match the existing project when importing."
  type        = string
  default     = "ArmadaCMS"
}

variable "project_region" {
  description = "Supabase region of the production project. Must match the existing project when importing."
  type        = string
  default     = "eu-north-1"
}

variable "database_password" {
  description = "Current production database password for the imported Supabase project. The provider requires this in configuration, but the Management API does not return it back on import."
  type        = string
  sensitive   = true
}

variable "database_name" {
  description = "Primary PostgreSQL database name exposed to ArmadaCMS."
  type        = string
  default     = "postgres"
}

variable "database_user" {
  description = "Primary PostgreSQL database user exposed to ArmadaCMS."
  type        = string
  default     = "postgres"
}

variable "api_db_schema" {
  description = "Comma-separated list of schemas exposed through the Supabase Data API / PostgREST configuration. Leave empty to keep the existing project setting unmanaged."
  type        = string
  default     = ""
}

variable "api_db_extra_search_path" {
  description = "Comma-separated Postgres search path used by the Data API."
  type        = string
  default     = "public,extensions"
}

variable "api_max_rows" {
  description = "Maximum number of rows returned by the Data API in a single response."
  type        = number
  default     = 1000
}

variable "pooler_host" {
  description = "Supabase transaction pooler host for ArmadaCMS (e.g. aws-1-eu-north-1.pooler.supabase.com). Not derivable from project_ref — set in prod.auto.tfvars and exported so gcp/prod can read it via tfe_outputs."
  type        = string
}

# ── Staging branch DB connection ──────────────────────────────────────────────
# Staging is a branch of the same Supabase project. GCP staging can connect directly to the staging branch DB via IPv6, so no pooler is needed.
# The staging branch connection details are stored as variables here and exported so gcp/staging can read them via tfe_outputs without hardcoding.

variable "staging_db_host" {
  description = "Direct PostgreSQL host for the staging Supabase branch. Set in prod.auto.tfvars and exported so gcp/staging can read it via tfe_outputs."
  type        = string
}

variable "staging_db_user" {
  description = "PostgreSQL user for the staging Supabase branch."
  type        = string
  default     = "postgres"
}

variable "staging_db_name" {
  description = "PostgreSQL database name for the staging Supabase branch."
  type        = string
  default     = "postgres"
}

# ── Supabase Storage ─────────────────────────────────────────────────────────────
# Exported so gcp/prod and gcp/staging can read them via tfe_outputs,
# keeping storage configuration in one place.

variable "supabase_url" {
  description = "Base HTTPS URL of the Supabase project (e.g. https://<ref>.supabase.co). Exported to GCP workspaces via tfe_outputs."
  type        = string
  default     = "https://rsdjnixgxqauonaofrwr.supabase.co"
}

variable "supabase_storage_s3_endpoint" {
  description = "S3-compatible Supabase Storage endpoint for server-side uploads (e.g. https://<ref>.storage.supabase.co/storage/v1/s3). Exported to GCP workspaces via tfe_outputs."
  type        = string
  default     = "https://rsdjnixgxqauonaofrwr.storage.supabase.co/storage/v1/s3"
}

variable "supabase_storage_bucket" {
  description = "Supabase Storage bucket used by ArmadaCMS for file uploads. Exported to GCP workspaces via tfe_outputs."
  type        = string
  default     = "armadacms-files"
}

variable "supabase_storage_region" {
  description = "AWS-style region name reported to the S3-compatible client (e.g. eu-north-1). Exported to GCP workspaces via tfe_outputs."
  type        = string
  default     = "eu-north-1"
}
