# Import blocks for existing AWS resources.
# Run `terraform plan` after `terraform init` — Terraform will show what it
# would change. Resources that are already in sync will show "No changes."
#
# Remove a block after its resource has been imported and the plan is clean,
# or leave it in place — Terraform is idempotent on already-imported resources.

# ── RDS ────────────────────────────────────────────────────────────────────────

import {
  id = "sg-09a9bc7295247ba27"
  to = aws_security_group.rds
}

import {
  id = local.rds_identifier
  to = aws_db_instance.main
}

# ── S3 ─────────────────────────────────────────────────────────────────────────

import {
  id = local.s3_bucket_name
  to = aws_s3_bucket.cms_files
}

import {
  id = local.s3_bucket_name
  to = aws_s3_bucket_public_access_block.cms_files
}

import {
  id = local.s3_bucket_name
  to = aws_s3_bucket_policy.cms_files
}

# ── IAM ────────────────────────────────────────────────────────────────────────

import {
  id = "arn:aws:iam::593054043164:policy/ArmadaCMSProdS3UploadsPolicy"
  to = aws_iam_policy.s3_uploads
}

import {
  id = "ArmadaCMSProductionUploads"
  to = aws_iam_group.s3_uploaders
}

import {
  id = "ArmadaCMSProductionUploads/arn:aws:iam::593054043164:policy/ArmadaCMSProdS3UploadsPolicy"
  to = aws_iam_group_policy_attachment.s3_uploads
}

import {
  id = "armadacms-prod-s3"
  to = aws_iam_user.s3_uploader
}

import {
  id = "armadacms-prod-s3/ArmadaCMSProductionUploads"
  to = aws_iam_user_group_membership.s3_uploader
}
