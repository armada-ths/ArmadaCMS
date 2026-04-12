# ── IAM policy ─────────────────────────────────────────────────────────────────

resource "aws_iam_policy" "s3_uploads" {
  name = "ArmadaCMSStagingS3UploadsPolicy"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "AllowWriteToStagingBucket"
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.cms_files.arn}/*"
      },
    ]
  })

  tags = local.common_tags
}

# ── IAM group ──────────────────────────────────────────────────────────────────

resource "aws_iam_group" "s3_uploaders" {
  name = "ArmadaCMSStagingUploads"
}

resource "aws_iam_group_policy_attachment" "s3_uploads" {
  group      = aws_iam_group.s3_uploaders.name
  policy_arn = aws_iam_policy.s3_uploads.arn
}

# ── IAM user ───────────────────────────────────────────────────────────────────

resource "aws_iam_user" "s3_uploader" {
  name = "armadacms-staging-s3"

  tags = local.common_tags
}

resource "aws_iam_user_group_membership" "s3_uploader" {
  user   = aws_iam_user.s3_uploader.name
  groups = [aws_iam_group.s3_uploaders.name]
}

# ── Access key ─────────────────────────────────────────────────────────────────
# The access key for armadacms-staging-s3 must be created manually in the AWS
# console and stored as GCP Secret Manager secrets:
#   armadacms-staging-AWS_ACCESS_KEY_ID
#   armadacms-staging-AWS_SECRET_ACCESS_KEY
# Terraform-managed access keys store the secret in state, which creates
# unnecessary exposure.
