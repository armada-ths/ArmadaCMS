# ── IAM policy ─────────────────────────────────────────────────────────────────

resource "aws_iam_policy" "s3_uploads" {
  name        = "ArmadaCMSProdS3UploadsPolicy"
  description = "Allows the ArmadaCMS Cloud Run service to upload files to the production S3 bucket. Write access is restricted to the Cloud Run NAT outbound IP."

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "AllowWriteToProdBucket"
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.cms_files.arn}/*"
      },
      {
        Sid      = "DenyWriteOutsideTrustedIp"
        Effect   = "Deny"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.cms_files.arn}/*"
        Condition = {
          NotIpAddress = {
            "aws:SourceIp" = local.nat_cidr
          }
        }
      },
    ]
  })

  tags = local.common_tags
}

# ── IAM group ──────────────────────────────────────────────────────────────────

resource "aws_iam_group" "s3_uploaders" {
  name = "ArmadaCMSProductionUploads"
}

resource "aws_iam_group_policy_attachment" "s3_uploads" {
  group      = aws_iam_group.s3_uploaders.name
  policy_arn = aws_iam_policy.s3_uploads.arn
}

# ── IAM user ───────────────────────────────────────────────────────────────────

resource "aws_iam_user" "s3_uploader" {
  name = "armadacms-prod-s3"

  tags = local.common_tags
}

resource "aws_iam_group_membership" "s3_uploader" {
  name  = "armadacms-prod-s3-membership"
  group = aws_iam_group.s3_uploaders.name
  users = [aws_iam_user.s3_uploader.name]
}

# ── Access key ─────────────────────────────────────────────────────────────────
# The access key for armadacms-prod-s3 was created outside Terraform and is NOT
# managed here. Terraform-managed access keys store the secret in state, which
# creates unnecessary exposure. Rotate the key manually in the AWS console and
# update the corresponding secrets in Cloud Run / Secret Manager as needed.
