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

variable "cloud_run_nat_ip_override" {
  description = "Override for the Cloud Run NAT outbound IP used in network restrictions. Leave null to read automatically from the armadacms-gcp-prod workspace output 'static_egress_ip'."
  type        = string
  default     = null
}

# ── Staging branch DB connection ──────────────────────────────────────────────
# Staging is a branch of the same Supabase project. GCP staging can connect directly to the staging branch DB via IPv6, so no pooler is needed.
# The staging branch connection details are stored as variables here and exported so gcp/staging can read them via tfe_outputs without hardcoding.

variable "staging_project_ref" {
  description = "Supabase project reference for the staging branch. Used to manage staging-specific settings (auth, PostgREST) and to derive staging_db_host (db.<ref>.supabase.co)."
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
