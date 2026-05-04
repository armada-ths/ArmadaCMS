import {
  to = supabase_settings.production
  id = var.project_ref
}

resource "supabase_settings" "production" {
  project_ref = var.project_ref

  api = jsonencode(local.managed_api_settings)

  lifecycle {
    ignore_changes = [api]
  }
}

data "supabase_pooler" "production" {
  project_ref = var.project_ref
}
