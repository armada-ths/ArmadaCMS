output "project_ref" {
  description = "Supabase project reference for ArmadaCMS production."
  value       = var.project_ref
}

output "project_name" {
  description = "Managed Supabase project name."
  value       = supabase_project.production.name
}

output "project_region" {
  description = "Region of the managed Supabase production project."
  value       = supabase_project.production.region
}

output "dashboard_project_url" {
  description = "Supabase dashboard URL for the production project."
  value       = local.dashboard_project_url
}

output "db_host" {
  description = "Direct PostgreSQL host for ArmadaCMS production."
  value       = local.direct_db_host
}

output "db_name" {
  description = "Primary PostgreSQL database name for ArmadaCMS production."
  value       = var.database_name
}

output "db_user" {
  description = "Primary PostgreSQL user for ArmadaCMS production."
  value       = var.database_user
}

output "pooler_host" {
  description = "Supabase transaction pooler host consumed by gcp/prod via tfe_outputs."
  value       = var.pooler_host
}

output "pooler_user" {
  description = "Supabase transaction pooler user (postgres.<project_ref>) consumed by gcp/prod via tfe_outputs."
  value       = local.pooler_user
}

output "staging_db_host" {
  description = "Direct PostgreSQL host for the staging Supabase branch, consumed by gcp/staging via tfe_outputs."
  value       = var.staging_db_host
}

output "staging_db_user" {
  description = "PostgreSQL user for the staging Supabase branch."
  value       = var.staging_db_user
}

output "staging_db_name" {
  description = "PostgreSQL database name for the staging Supabase branch."
  value       = var.staging_db_name
}

output "pooler_urls" {
  description = "Supabase pooler connection strings keyed by pool mode. Sensitive because they may embed credentials depending on account configuration."
  value       = data.supabase_pooler.production.url
  sensitive   = true
}

output "managed_api_settings" {
  description = "Desired Supabase API settings for future Terraform management. Currently informational only because this root does not actively manage supabase_settings."
  value       = local.managed_api_settings
}
