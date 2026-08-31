project_id   = "just-sunrise-491718-m9"
region       = "europe-north1"
environment  = "staging"
service_name = "armadacms-staging"
name_prefix  = "armadacms-staging"

deploy_cloud_run_service = true
deletion_protection      = false

cloud_run_ingress = "INGRESS_TRAFFIC_ALL"

manage_runtime_service_account     = true
manage_cloud_build_service_account = true
manage_secret_accessor_bindings    = true
manage_cloud_build_triggers        = true

cloud_run_cpu    = "1000m"
cloud_run_memory = "512Mi"
min_instances    = 0
max_instances    = 1

enable_vpc_egress = false

enable_domain_mapping   = true
domain_mapping_hostname = "staging.cms.armada.nu"

revalidation_url = "https://staging.armada.nu/api/revalidate"

manage_github_app_secret = false

enable_https_load_balancer = false

enable_recaptcha          = true
recaptcha_allowed_domains = ["staging.armada.nu"]
