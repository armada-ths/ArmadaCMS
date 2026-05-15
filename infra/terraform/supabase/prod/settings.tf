data "supabase_pooler" "production" {
  project_ref = var.project_ref
}

# supabase_settings is intentionally NOT managed here.
#
# The provider performs a REST-service health probe before applying any
# supabase_settings resource. On this project that probe can false-fail even
# while the dashboard shows the project as healthy, causing the apply to error.
# This is a known provider limitation unrelated to the settings values.
