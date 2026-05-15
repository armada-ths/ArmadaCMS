data "supabase_pooler" "production" {
  project_ref = var.project_ref
}

# ── Production project settings ──────────────────────────────────────────────
#
# ArmadaCMS connects directly to Postgres via its Go API and does not use
# Supabase Auth, PostgREST, or Edge Functions. These settings lock down the
# services that are not needed.

resource "supabase_settings" "production" {
  project_ref = var.project_ref

  # Disable new user sign-ups through Supabase Auth. ArmadaCMS uses its own
  # HS256 JWT auth and never calls the Supabase Auth endpoints.
  auth = jsonencode({
    disable_signup = true
  })

  # Expose no schemas through the PostgREST Data API. Setting db_schema to
  # an empty string disables REST/GraphQL data access without touching the
  # underlying Postgres schema. Provider v1.9.1 handles this state correctly.
  api = jsonencode({
    db_schema            = ""
    db_extra_search_path = ""
  })
}

# ── Staging branch settings ───────────────────────────────────────────────────
#
# The staging branch is a separate Supabase project (its own project_ref)
# branched from the production project. PostgREST is disabled here too,
# and auth signup is locked down to match production.

resource "supabase_settings" "staging" {
  project_ref = var.staging_project_ref

  auth = jsonencode({
    disable_signup = true
  })

  api = jsonencode({
    db_schema            = ""
    db_extra_search_path = ""
  })
}
