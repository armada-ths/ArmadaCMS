locals {
  dashboard_project_url = "https://supabase.com/dashboard/project/${var.project_ref}"
  direct_db_host        = "db.${var.project_ref}.supabase.co"

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
