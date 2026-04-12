locals {
  common_tags = {
    Application = "ArmadaCMS"
    Environment = var.environment
    ManagedBy   = "terraform"
    Repository  = "armadacms"
  }

  s3_bucket_name = "armada-cms-files-staging-b3f79a2e1d84"
}
