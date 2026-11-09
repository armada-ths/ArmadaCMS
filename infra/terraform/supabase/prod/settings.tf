data "supabase_pooler" "production" {
  project_ref = var.project_ref
}

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

# Staging settings target the persistent branch's project ref.
resource "supabase_settings" "staging" {
  project_ref = var.staging_project_ref

  api             = jsonencode(local.managed_api_settings)
  auth            = jsonencode(local.managed_auth_settings)
  storage         = jsonencode(local.managed_storage_settings)
  ssl_enforcement = true

  # Staging uses a direct IPv6 connection without a static Cloud NAT IP.
  network = jsonencode({
    restrictions = ["0.0.0.0/0", "::/0"]
  })
}
