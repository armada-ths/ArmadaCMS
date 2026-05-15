resource "supabase_project" "production" {
  organization_id   = var.organization_id
  name              = var.project_name
  database_password = var.database_password
  region            = var.project_region

  lifecycle {
    prevent_destroy = true
  }
}
