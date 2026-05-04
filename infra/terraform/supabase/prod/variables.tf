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
  description = "Comma-separated list of schemas exposed through the Supabase Data API / PostgREST configuration."
  type        = string
  default     = "public,storage,graphql_public"
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
