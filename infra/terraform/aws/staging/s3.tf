resource "aws_s3_bucket" "cms_files" {
  bucket = local.s3_bucket_name

  tags = merge(local.common_tags, {
    Name = local.s3_bucket_name
  })
}

# Public ACLs are disabled at the block level; public read is granted through
# the bucket policy only (PublicReadGetObject statement below).
resource "aws_s3_bucket_public_access_block" "cms_files" {
  bucket = aws_s3_bucket.cms_files.id

  block_public_acls       = false
  ignore_public_acls      = false
  block_public_policy     = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "cms_files" {
  bucket = aws_s3_bucket.cms_files.id

  # Allow unauthenticated GetObject (public file serving) and deny any request
  # that does not use HTTPS.
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.cms_files.arn}/*"
      },
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.cms_files.arn,
          "${aws_s3_bucket.cms_files.arn}/*",
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      },
    ]
  })

  depends_on = [aws_s3_bucket_public_access_block.cms_files]
}
