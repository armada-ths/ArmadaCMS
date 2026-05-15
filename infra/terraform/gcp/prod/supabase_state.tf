# Read outputs from the Supabase production workspace so DB_HOST and DB_USER
# are always in sync without manual updates in this workspace.
#
# The tfe provider authenticates using the same HCP Terraform token that
# runs this workspace — no extra credentials are needed.
# The Supabase workspace must grant read access to this workspace under
# Settings → Remote state sharing (or "Share with all workspaces").

data "tfe_outputs" "supabase_prod" {
  organization = "THS-Armada"
  workspace    = "armadacms-supabase-prod"
}
