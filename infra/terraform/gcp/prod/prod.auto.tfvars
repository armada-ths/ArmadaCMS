project_id   = "just-sunrise-491718-m9"
region       = "europe-north2"
environment  = "production"
service_name = "armadacms"
name_prefix  = "armadacms"

artifact_registry_repository_id = "cloud-run-source-deploy"
container_image_path            = "armadacms/armadacms"
bootstrap_image_tag             = "bootstrap"

deploy_cloud_run_service = true
allow_unauthenticated    = false
deletion_protection      = true

manage_runtime_service_account    = false
cloud_run_service_account_email   = "475154911163-compute@developer.gserviceaccount.com"
cloud_build_service_account_email = "475154911163-compute@developer.gserviceaccount.com"
manage_cloud_build_triggers       = true

cloud_run_cpu             = "1000m"
cloud_run_memory          = "1Gi"
cloud_run_timeout_seconds = 120
cloud_run_concurrency     = 10
min_instances             = 0
max_instances             = 2

enable_vpc_egress            = true
cloud_run_vpc_egress_mode    = "DIRECT_VPC"
manage_vpc_network_resources = false
vpc_network_name             = "default"
vpc_subnetwork_name          = "default"
vpc_subnet_cidr              = "10.226.0.0/20"
vpc_connector_name           = "armadacms-serverless"
vpc_connector_cidr           = "10.8.0.0/28"
vpc_egress                   = "ALL_TRAFFIC"
nat_router_name              = "armadacms-router"
nat_name                     = "armadacms-nat"
nat_ip_name                  = "armadacms-nat-ip"

enable_https_load_balancer     = true
lb_reserve_global_ip           = true
lb_enable_http_redirect        = true
lb_managed_certificate_domains = ["cms.armada.nu"]

additional_plain_env_vars = {
  DB_HOST                       = "terraform-20250411194126172200000001.cg6cp6bnc2ao.eu-north-1.rds.amazonaws.com"
  DB_PORT                       = "5432"
  DB_USER                       = "postgres"
  DB_NAME                       = "armadacms"
  DB_SSLMODE                    = "require"
  S3_BUCKET                     = "armada-cms-files-e48105192c52"
  AWS_REGION                    = "eu-north-1"
  DB_MAX_OPEN_CONNS             = "10"
  DB_MAX_IDLE_CONNS             = "5"
  DB_CONN_MAX_LIFETIME_MINUTES  = "30"
  DB_CONN_MAX_IDLE_TIME_MINUTES = "10"
  AUDIT_LOG_RETENTION_DAYS      = "7"
}
