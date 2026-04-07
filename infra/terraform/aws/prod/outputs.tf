output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint (host:port)."
  value       = "${aws_db_instance.main.address}:${aws_db_instance.main.port}"
}

output "rds_host" {
  description = "RDS PostgreSQL hostname (without port). Used by the GCP workspace to configure Cloud Run's DB_HOST environment variable."
  value       = aws_db_instance.main.address
}

output "rds_db_name" {
  description = "PostgreSQL database name."
  value       = aws_db_instance.main.db_name
}

output "rds_security_group_id" {
  description = "Security group ID attached to the RDS instance."
  value       = aws_security_group.rds.id
}

output "s3_bucket_name" {
  description = "S3 bucket used for CMS file uploads."
  value       = aws_s3_bucket.cms_files.bucket
}

output "s3_bucket_arn" {
  description = "ARN of the S3 bucket."
  value       = aws_s3_bucket.cms_files.arn
}

output "s3_bucket_region" {
  description = "AWS region of the S3 bucket."
  value       = aws_s3_bucket.cms_files.region
}

output "iam_uploader_user_arn" {
  description = "ARN of the IAM user that holds the S3 upload credentials (armadacms-prod-s3)."
  value       = aws_iam_user.s3_uploader.arn
}

output "iam_uploads_policy_arn" {
  description = "ARN of the IAM policy that grants S3 write access to the upload user."
  value       = aws_iam_policy.s3_uploads.arn
}
