output "s3_bucket_name" {
  description = "S3 bucket used for CMS staging file uploads."
  value       = aws_s3_bucket.cms_files.bucket
}

output "s3_bucket_arn" {
  description = "ARN of the staging S3 bucket."
  value       = aws_s3_bucket.cms_files.arn
}

output "s3_bucket_region" {
  description = "AWS region of the staging S3 bucket."
  value       = aws_s3_bucket.cms_files.region
}

output "iam_uploader_user_arn" {
  description = "ARN of the IAM user that holds the staging S3 upload credentials (armadacms-staging-s3)."
  value       = aws_iam_user.s3_uploader.arn
}

output "iam_uploads_policy_arn" {
  description = "ARN of the IAM policy that grants S3 write access to the staging upload user."
  value       = aws_iam_policy.s3_uploads.arn
}
