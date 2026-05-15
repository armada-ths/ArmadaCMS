locals {
  dashboard_project_url = "https://supabase.com/dashboard/project/${var.project_ref}"
  direct_db_host        = "db.${var.project_ref}.supabase.co"

  # Pooler user follows the Supabase convention: postgres.<project_ref>
  pooler_user = "postgres.${var.project_ref}"

  managed_api_settings = merge(
    {
      db_extra_search_path = var.api_db_extra_search_path
      max_rows             = var.api_max_rows
    },
    trimspace(var.api_db_schema) != "" ? {
      db_schema = var.api_db_schema
    } : {}
  )
}
