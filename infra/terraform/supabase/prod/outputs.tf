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

output "pooler_urls" {
  description = "Supabase pooler connection strings keyed by pool mode. Sensitive because they may embed credentials depending on account configuration."
  value       = data.supabase_pooler.production.url
  sensitive   = true
}

output "managed_api_settings" {
  description = "Desired Supabase API settings for future Terraform management. Currently informational only because this root does not actively manage supabase_settings."
  value       = local.managed_api_settings
}
