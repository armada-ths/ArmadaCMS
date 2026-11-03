data "supabase_pooler" "production" {
  project_ref = var.project_ref
}

# Adopt the existing hosted settings. Only the JSON keys below are managed;
# credentials and other service configuration remain outside this root.
resource "supabase_settings" "production" {
  project_ref = var.project_ref

  api             = jsonencode(local.managed_api_settings)
  auth            = jsonencode(local.managed_auth_settings)
  storage         = jsonencode(local.managed_storage_settings)
  ssl_enforcement = true

  network = jsonencode({
    restrictions = local.production_network_restrictions
  })

  lifecycle {
    precondition {
      condition     = can(cidrnetmask("${local.gcp_prod_egress_ip}/32"))
      error_message = "armadacms-gcp-prod must export a valid static_egress_ip before production network restrictions can be managed."
    }
  }
}

# The persistent branch has its own project ref. Provider 1.11.0 resolves
# branch refs through the branches endpoint when the projects endpoint is 404.
resource "supabase_settings" "staging" {
  project_ref = var.staging_project_ref

  api             = jsonencode(local.managed_api_settings)
  auth            = jsonencode(local.managed_auth_settings)
  storage         = jsonencode(local.managed_storage_settings)
  ssl_enforcement = true

  # Preserve staging's current IPv4/IPv6 access; it has no static Cloud NAT IP.
  network = jsonencode({
    restrictions = ["0.0.0.0/0", "::/0"]
  })
}
