region      = "eu-north-1"
environment = "production"

# cloud_run_nat_ip is the static outbound IP provisioned in gcp/prod/networking.tf.
# Update this if the Cloud Run NAT IP ever changes — it controls both the RDS
# security group ingress rule and the IAM policy IP condition for S3 writes.
cloud_run_nat_ip = "34.51.249.94"
