locals {
  common_tags = {
    Application = "ArmadaCMS"
    Environment = var.environment
    ManagedBy   = "terraform"
    Repository  = "armadacms"
  }

  s3_bucket_name = "armada-cms-files-e48105192c52"
  rds_identifier = "terraform-20250411194126172200000001"
  nat_cidr       = "${coalesce(var.cloud_run_nat_ip_override, data.tfe_outputs.gcp_prod.values["static_egress_ip"])}/32"
  vpc_id         = "vpc-0fd35de47c01b2187"
}
