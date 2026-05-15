# Restrict direct PostgreSQL access (port 5432) to the Cloud Run NAT IP only.
#
# The allowed CIDR is read automatically from the armadacms-gcp-prod workspace
# output 'static_egress_ip'. Set cloud_run_nat_ip_override if you need to
# temporarily use a different IP (e.g. during a NAT failover).
#
# Applies to direct database connections only — the REST API and Supabase
# dashboard access are unaffected by this restriction.

resource "supabase_network_restrictions" "production" {
  project_ref     = var.project_ref
  db_allowed_cidr = [local.nat_cidr]
}
