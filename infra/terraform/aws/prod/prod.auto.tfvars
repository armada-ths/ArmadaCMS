region      = "eu-north-1"
environment = "production"

# cloud_run_nat_ip_override is intentionally omitted — the NAT IP is read
# automatically from the armadacms-gcp-prod workspace output "static_egress_ip".
# Set this only if you need to bypass the remote state lookup:
# cloud_run_nat_ip_override = "1.2.3.4"
