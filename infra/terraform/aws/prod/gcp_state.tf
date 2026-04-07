# Read outputs from the GCP production workspace so the Cloud Run NAT IP
# is always in sync without manual updates.
#
# The tfe provider authenticates using the same HCP Terraform token that
# runs this workspace — no extra credentials are needed.
# The GCP workspace must have "Share state globally" or grant read access
# to this workspace under Settings → Remote state sharing.

data "tfe_outputs" "gcp_prod" {
  organization = "THS-Armada"
  workspace    = "armadacms-gcp-prod"
}
