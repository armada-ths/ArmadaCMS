# Read outputs from the Supabase production workspace so staging DB_HOST,
# DB_USER, and Supabase Storage configuration are always in sync without
# manual updates in this workspace.
#
# Staging is a branch of the same Supabase project managed by armadacms-supabase-prod.
# The staging-specific outputs (staging_db_host, staging_db_user, staging_db_name)
# are defined there alongside the production pooler values and storage outputs.
#
# The tfe provider authenticates using the same HCP Terraform token that
# runs this workspace — no extra credentials are needed.
# The Supabase workspace must grant read access to this workspace under
# Settings → Remote state sharing (or "Share with all workspaces").

data "tfe_outputs" "supabase_prod" {
  organization = "THS-Armada"
  workspace    = "armadacms-supabase-prod"
}
