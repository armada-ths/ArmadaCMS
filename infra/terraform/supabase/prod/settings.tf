data "supabase_pooler" "production" {
  project_ref = var.project_ref
}

# ── Production project settings ──────────────────────────────────────────────────
#
# ArmadaCMS does not use Supabase Auth, PostgREST, or Edge Functions.
# These settings lock down unused services and restrict DB network access
# to the Cloud Run NAT IP.
#
# The provider performs a REST-service health probe before applying changes.
# If the apply times out, re-run — the probe is known to be slow on imported
# projects. A 15-minute timeout is set to give it sufficient time.
#
# Realtime cannot be managed via supabase_settings (no realtime block in the
# provider). Disable it manually in the dashboard if needed.

resource "supabase_settings" "production" {
  project_ref = var.project_ref

  timeouts {
    create = "15m"
    update = "15m"
  }

  # Restrict direct PostgreSQL access to the Cloud Run NAT IP only.
  # The provider automatically calls the /apply endpoint after updating.
  network = jsonencode({
    restrictions = [local.nat_cidr]
  })

  # Disable Supabase Auth signup. ArmadaCMS uses its own HS256 JWT auth.
  auth = jsonencode({
    disable_signup = true
  })

  # Expose no schemas through PostgREST. ArmadaCMS connects directly via
  # Go/GORM and never uses the Data API.
  api = jsonencode({
    db_schema            = ""
    db_extra_search_path = ""
  })
}

# ── Staging branch settings ──────────────────────────────────────────────────
#
# Staging uses a direct IPv6 connection from GCP, so no network CIDR
# restriction is added here (IPv6 egress is not a fixed IP).

resource "supabase_settings" "staging" {
  project_ref = var.staging_project_ref

  timeouts {
    create = "15m"
    update = "15m"
  }

  auth = jsonencode({
    disable_signup = true
  })

  api = jsonencode({
    db_schema            = ""
    db_extra_search_path = ""
  })
}
