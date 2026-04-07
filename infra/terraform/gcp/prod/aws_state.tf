# Read outputs from the AWS production workspace so DB_HOST is always
# in sync without manual updates.
#
# The tfe provider authenticates using the same HCP Terraform token that
# runs this workspace — no extra credentials needed.
# The AWS workspace must grant read access to this workspace under
# Settings → Remote state sharing (or "Share with all workspaces").

data "tfe_outputs" "aws_prod" {
  organization = "THS-Armada"
  workspace    = "armadacms-aws-prod"
}
