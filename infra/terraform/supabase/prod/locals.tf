locals {
  dashboard_project_url = "https://supabase.com/dashboard/project/${var.project_ref}"
  direct_db_host        = "db.${var.project_ref}.supabase.co"

  # Pooler user follows the Supabase convention: postgres.<project_ref>
  pooler_user = "postgres.${var.project_ref}"

  gcp_prod_egress_ip              = try(nonsensitive(data.tfe_outputs.gcp_prod.values["static_egress_ip"]), null)
  production_network_restrictions = local.gcp_prod_egress_ip != null ? ["${local.gcp_prod_egress_ip}/32"] : []

  managed_api_settings = {
    db_schema            = var.api_db_schema
    db_extra_search_path = var.api_db_extra_search_path
    max_rows             = var.api_max_rows
  }

  # ArmadaCMS authenticates through its Go API, not Supabase Auth.
  managed_auth_settings = {
    disable_signup = true
  }

  managed_storage_settings = {
    fileSizeLimit = var.storage_file_size_limit
    features = {
      imageTransformation = { enabled = true }
      s3Protocol          = { enabled = true }
    }
  }
}
