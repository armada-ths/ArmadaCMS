# Read outputs from the AWS staging workspace so S3_BUCKET and AWS_REGION are
# always in sync without manual updates.
#
# The tfe provider authenticates using the same HCP Terraform token that runs
# this workspace — no extra credentials needed.
# The AWS staging workspace must grant read access to this workspace under
# Settings → Remote state sharing (or "Share with all workspaces").
#
# BOOTSTRAPPING ORDER: Apply aws/staging first (it has no dependency on this
# workspace), then apply gcp/staging with deploy_cloud_run_service = true once
# the S3 outputs are available.

data "tfe_outputs" "aws_staging" {
  organization = "THS-Armada"
  workspace    = "armadacms-aws-staging"
}
